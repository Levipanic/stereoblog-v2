package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const migrationTableSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TEXT NOT NULL DEFAULT (datetime('now'))
)`

const migrationTableSignature = "version:INTEGER:0:-:1,name:TEXT:1:-:0,applied_at:TEXT:1:datetime('now'):0"

const v1BaselineSQL = `
CREATE TABLE IF NOT EXISTS posts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  blocks_json TEXT NOT NULL,
  likes_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  preview_media TEXT DEFAULT NULL
);
CREATE TABLE IF NOT EXISTS comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_id INTEGER NOT NULL,
  parent_id INTEGER,
  name TEXT,
  content TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  likes_count INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'visible',
  moderation_reason TEXT,
  text_hash TEXT,
  text_fingerprint TEXT,
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
  FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_comments_post_parent ON comments(post_id, parent_id, id);
CREATE INDEX IF NOT EXISTS idx_comments_status_created ON comments(status, created_at);
CREATE TABLE IF NOT EXISTS like_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_id INTEGER NOT NULL,
  ip_hash TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_like_events_post_ip_created ON like_events(post_id, ip_hash, created_at);
CREATE TABLE IF NOT EXISTS comment_like_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  comment_id INTEGER NOT NULL,
  ip_hash TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_comment_like_events_comment_ip_created ON comment_like_events(comment_id, ip_hash, created_at);
CREATE TABLE IF NOT EXISTS comment_attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ip_hash TEXT NOT NULL,
  post_id INTEGER,
  status TEXT NOT NULL,
  reason TEXT,
  content TEXT,
  text_hash TEXT,
  fingerprint TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_comment_attempts_ip_time ON comment_attempts(ip_hash, created_at);
CREATE INDEX IF NOT EXISTS idx_comment_attempts_post_time ON comment_attempts(post_id, created_at);
CREATE INDEX IF NOT EXISTS idx_comment_attempts_ip_post_hash_time ON comment_attempts(ip_hash, post_id, text_hash, created_at);
CREATE INDEX IF NOT EXISTS idx_comment_attempts_ip_status_time ON comment_attempts(ip_hash, status, created_at);
CREATE TABLE IF NOT EXISTS comment_mutes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ip_hash TEXT NOT NULL UNIQUE,
  reason TEXT,
  muted_until TEXT NOT NULL,
  mute_count INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_comment_mutes_until ON comment_mutes(muted_until);
CREATE TABLE IF NOT EXISTS comment_challenge_uses (
  token_hash TEXT PRIMARY KEY,
  post_id INTEGER NOT NULL,
  used_count INTEGER NOT NULL DEFAULT 0,
  first_used_at TEXT NOT NULL DEFAULT (datetime('now')),
  last_used_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS admin_sessions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  token_hash TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_admin_sessions_expires_at ON admin_sessions(expires_at);`

type migration struct {
	version int
	name    string
	sql     string
	apply   func(context.Context, *sql.Conn) error
}

var migrations = []migration{
	{version: 1, name: "v1_baseline", sql: v1BaselineSQL},
	{version: 2, name: "post_slugs", apply: addPostSlugs},
}

type MigrationResult struct {
	Version    int
	Applied    int
	BackupPath string
}

func Migrate(ctx context.Context, db *sql.DB, databasePath string, schema Schema, production bool) (MigrationResult, error) {
	applied, err := appliedMigrations(ctx, db, migrations)
	if err != nil {
		return MigrationResult{}, err
	}
	if err := validateMigratedSchema(ctx, db, applied); err != nil {
		return MigrationResult{}, err
	}
	result := MigrationResult{Version: len(applied)}
	if len(applied) == len(migrations) {
		return result, nil
	}
	if production && schema != SchemaFresh {
		result.BackupPath, err = backupDatabase(ctx, db, databasePath, time.Now().UTC())
		if err != nil {
			return MigrationResult{}, fmt.Errorf("pre-migration backup: %w", err)
		}
	}
	result.Applied, err = applyMigrations(ctx, db, migrations)
	if err != nil {
		if result.BackupPath != "" {
			return MigrationResult{}, fmt.Errorf("migrate database (backup retained at %s): %w", result.BackupPath, err)
		}
		return MigrationResult{}, err
	}
	result.Version += result.Applied
	return result, nil
}

func appliedMigrations(ctx context.Context, db queryer, available []migration) ([]migration, error) {
	for index, migration := range available {
		if migration.version != index+1 || migration.name == "" {
			return nil, fmt.Errorf("invalid compiled migration at position %d", index+1)
		}
	}
	var exists int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name = 'schema_migrations'").Scan(&exists); err != nil {
		return nil, fmt.Errorf("inspect migration state: %w", err)
	}
	if exists == 0 {
		return nil, nil
	}
	signature, err := columnSignature(ctx, db, "schema_migrations")
	if err != nil {
		return nil, err
	}
	if signature != migrationTableSignature {
		return nil, errors.New("unsupported migration table schema")
	}
	rows, err := db.QueryContext(ctx, "SELECT version, name FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("inspect migration state: %w", err)
	}
	defer rows.Close()
	var applied []migration
	for rows.Next() {
		var version int
		var name string
		if err := rows.Scan(&version, &name); err != nil {
			return nil, fmt.Errorf("inspect migration state: %w", err)
		}
		if version != len(applied)+1 || version > len(available) || available[version-1].name != name {
			return nil, fmt.Errorf("unsupported migration state at version %d", version)
		}
		applied = append(applied, available[version-1])
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inspect migration state: %w", err)
	}
	return applied, nil
}

func applyMigrations(ctx context.Context, db *sql.DB, available []migration) (applied int, err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return 0, fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return 0, fmt.Errorf("begin migrations: %w", err)
	}
	defer func() {
		if err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	completed, err := appliedMigrations(ctx, conn, available)
	if err != nil {
		return 0, err
	}
	if _, err = conn.ExecContext(ctx, migrationTableSQL); err != nil {
		return 0, fmt.Errorf("create migration table: %w", err)
	}
	for _, migration := range available[len(completed):] {
		if migration.apply != nil {
			err = migration.apply(ctx, conn)
		} else {
			_, err = conn.ExecContext(ctx, migration.sql)
		}
		if err != nil {
			return applied, fmt.Errorf("apply migration %d (%s): %w", migration.version, migration.name, err)
		}
		if _, err = conn.ExecContext(ctx, "INSERT INTO schema_migrations (version, name) VALUES (?, ?)", migration.version, migration.name); err != nil {
			return applied, fmt.Errorf("record migration %d (%s): %w", migration.version, migration.name, err)
		}
		applied++
	}
	if err = validateMigratedSchema(ctx, conn, available); err != nil {
		return applied, err
	}
	if err = integrityCheck(ctx, conn); err != nil {
		return applied, fmt.Errorf("verify migrated database: %w", err)
	}
	foreignKeyRows, err := conn.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return applied, fmt.Errorf("verify migrated foreign keys: %w", err)
	}
	violated := foreignKeyRows.Next()
	rowsErr := foreignKeyRows.Err()
	if closeErr := foreignKeyRows.Close(); closeErr != nil {
		return applied, fmt.Errorf("verify migrated foreign keys: %w", closeErr)
	}
	if rowsErr != nil {
		return applied, fmt.Errorf("verify migrated foreign keys: %w", rowsErr)
	}
	if violated {
		return applied, errors.New("verify migrated foreign keys: foreign key check failed")
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return applied, fmt.Errorf("commit migrations: %w", err)
	}
	return applied, nil
}

func validateMigratedSchema(ctx context.Context, db queryer, applied []migration) error {
	if _, err := inspect(ctx, db); err != nil {
		return fmt.Errorf("verify migrated schema: %w", err)
	}
	if len(applied) < 2 {
		return nil
	}
	signature, err := columnSignature(ctx, db, "posts")
	if err != nil {
		return err
	}
	required := v1Signatures["posts"] + ",slug:TEXT:0:-:0"
	if signature != required && !strings.HasPrefix(signature, required+",") {
		return errors.New("verify migrated schema: incompatible posts.slug column")
	}
	index, exists, err := indexSignature(ctx, db, "posts", "idx_posts_slug")
	if err != nil {
		return err
	}
	if !exists || index != "1:0:slug" {
		return errors.New("verify migrated schema: incompatible slug index")
	}
	var missing int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM posts WHERE slug IS NULL OR slug = ''").Scan(&missing); err != nil {
		return fmt.Errorf("verify migrated slugs: %w", err)
	}
	if missing != 0 {
		return fmt.Errorf("verify migrated slugs: %d posts have no slug", missing)
	}
	var duplicates int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM (SELECT slug FROM posts GROUP BY slug HAVING count(*) > 1)").Scan(&duplicates); err != nil {
		return fmt.Errorf("verify migrated slug uniqueness: %w", err)
	}
	if duplicates != 0 {
		return fmt.Errorf("verify migrated slug uniqueness: %d duplicate slugs", duplicates)
	}
	return nil
}

func backupDatabase(ctx context.Context, db *sql.DB, databasePath string, now time.Time) (path string, err error) {
	absolutePath, err := filepath.Abs(databasePath)
	if err != nil {
		return "", fmt.Errorf("resolve database path: %w", err)
	}
	directory := filepath.Join(filepath.Dir(absolutePath), "backups")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return "", fmt.Errorf("secure backup directory: %w", err)
	}
	snapshotDirectory, err := os.MkdirTemp(directory, "pre-v2-migration-"+now.Format("20060102T150405.000000000Z")+"-")
	if err != nil {
		return "", fmt.Errorf("reserve backup directory: %w", err)
	}
	path = filepath.Join(snapshotDirectory, "blog.db")
	defer func() {
		if err != nil {
			_ = os.RemoveAll(snapshotDirectory)
		}
	}()
	if _, err = db.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		return "", fmt.Errorf("create SQLite snapshot: %w", err)
	}
	if err = os.Chmod(path, 0o600); err != nil {
		return "", fmt.Errorf("secure backup: %w", err)
	}
	if err = verifyBackup(ctx, path); err != nil {
		return "", err
	}
	return path, nil
}

func verifyBackup(ctx context.Context, path string) error {
	dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
	query := dsn.Query()
	query.Set("mode", "ro")
	query.Set("immutable", "1")
	dsn.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return fmt.Errorf("open backup: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if err := integrityCheck(ctx, db); err != nil {
		return fmt.Errorf("verify backup: %w", err)
	}
	return nil
}
