package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
)

func TestMigrateFreshDatabaseAndSkipCompletedMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blog.db")
	db, schema, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	result, err := Migrate(context.Background(), db, path, schema, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != len(migrations) || result.Applied != len(migrations) || result.BackupPath != "" {
		t.Fatalf("unexpected first migration result: %#v", result)
	}
	if migratedSchema, err := Inspect(context.Background(), db); err != nil || migratedSchema != SchemaV1 {
		t.Fatalf("migrated schema = %q, error = %v", migratedSchema, err)
	}

	result, err = Migrate(context.Background(), db, path, SchemaV1, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != len(migrations) || result.Applied != 0 || result.BackupPath != "" {
		t.Fatalf("unexpected repeated migration result: %#v", result)
	}
	var records int
	if err := db.QueryRow("SELECT count(*) FROM schema_migrations").Scan(&records); err != nil {
		t.Fatal(err)
	}
	if records != len(migrations) {
		t.Fatalf("migration records = %d, want %d", records, len(migrations))
	}
}

func TestMigrateV1PreservesDataAndIntegrity(t *testing.T) {
	path := testfixture.V1Database(t)
	db, schema, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	before := databaseContents(t, db)

	result, err := Migrate(context.Background(), db, path, schema, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != len(migrations) || result.Applied != len(migrations) || result.BackupPath != "" {
		t.Fatalf("unexpected migration result: %#v", result)
	}
	if after := databaseContents(t, db); !reflect.DeepEqual(after, before) {
		t.Fatalf("content changed\nbefore: %#v\nafter:  %#v", before, after)
	}
	assertDatabaseIntegrity(t, db)
}

func TestFailedMigrationRollsBackAndIsNotRecorded(t *testing.T) {
	t.Run("initial migration", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "fresh.db")
		db, _, err := Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		broken := []migration{{version: 1, name: "broken", sql: "CREATE TABLE must_rollback (id INTEGER); INSERT INTO must_rollback VALUES (1); invalid SQL"}}
		if _, err := applyMigrations(context.Background(), db, broken); err == nil {
			t.Fatal("broken initial migration succeeded")
		}
		var leaked int
		if err := db.QueryRow("SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name IN ('must_rollback', 'schema_migrations')").Scan(&leaked); err != nil {
			t.Fatal(err)
		}
		if leaked != 0 {
			t.Fatal("failed initial migration leaked schema changes")
		}
	})

	t.Run("later migration", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "blog.db")
		db, schema, err := Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if _, err := Migrate(context.Background(), db, path, schema, false); err != nil {
			t.Fatal(err)
		}

		broken := append([]migration{}, migrations...)
		broken = append(broken, migration{version: len(broken) + 1, name: "broken", sql: "CREATE TABLE must_rollback (id INTEGER); INSERT INTO must_rollback VALUES (1); invalid SQL"})
		if _, err := applyMigrations(context.Background(), db, broken); err == nil {
			t.Fatal("broken migration succeeded")
		}
		var table, version int
		if err := db.QueryRow("SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name = 'must_rollback'").Scan(&table); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT max(version) FROM schema_migrations").Scan(&version); err != nil {
			t.Fatal(err)
		}
		if table != 0 || version != len(migrations) {
			t.Fatalf("failed migration leaked changes: table=%d version=%d", table, version)
		}
	})
}

func TestProductionMigrationCreatesVerifiedPrivateBackup(t *testing.T) {
	path := testfixture.V1Database(t)
	db, schema, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	result, err := Migrate(context.Background(), db, path, schema, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath == "" {
		t.Fatal("production migration did not create a backup")
	}
	info, err := os.Stat(result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("backup permissions = %o, want 600", info.Mode().Perm())
	}
	directoryInfo, err := os.Stat(filepath.Dir(result.BackupPath))
	if err != nil {
		t.Fatal(err)
	}
	if directoryInfo.Mode().Perm() != 0o700 {
		t.Fatalf("backup directory permissions = %o, want 700", directoryInfo.Mode().Perm())
	}

	backup, err := sql.Open("sqlite", result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	if got := databaseContents(t, backup); !reflect.DeepEqual(got, databaseContents(t, db)) {
		t.Fatal("backup content differs from migrated source content")
	}
	var metadata int
	if err := backup.QueryRow("SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name = 'schema_migrations'").Scan(&metadata); err != nil {
		t.Fatal(err)
	}
	if metadata != 0 {
		t.Fatal("pre-migration backup contains post-backup migration metadata")
	}
	assertDatabaseIntegrity(t, backup)

	second, err := Migrate(context.Background(), db, path, SchemaV1, true)
	if err != nil {
		t.Fatal(err)
	}
	backups, err := filepath.Glob(filepath.Join(filepath.Dir(path), "backups", "pre-v2-migration-*", "blog.db"))
	if err != nil {
		t.Fatal(err)
	}
	if second.Applied != 0 || second.BackupPath != "" || len(backups) != 1 {
		t.Fatalf("repeat created another backup: result=%#v backups=%v", second, backups)
	}
}

func TestRejectsInvalidMigrationMetadata(t *testing.T) {
	for _, record := range []struct {
		version int
		name    string
	}{{1, "wrong_name"}, {len(migrations) + 1, "unknown"}} {
		t.Run(record.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.Exec(migrationTableSQL); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", record.version, record.name); err != nil {
				t.Fatal(err)
			}
			if _, err := appliedMigrations(context.Background(), db, migrations); err == nil {
				t.Fatal("invalid migration metadata was accepted")
			}
		})
	}
}

func TestRejectsMalformedMigrationTableAndCompiledOrder(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if _, err := appliedMigrations(context.Background(), db, migrations); err == nil {
		t.Fatal("malformed migration table was accepted")
	}
	if _, err := appliedMigrations(context.Background(), db, []migration{{version: 2, name: "out_of_order", sql: "SELECT 1"}}); err == nil {
		t.Fatal("out-of-order compiled migration was accepted")
	}
}

func databaseContents(t *testing.T, db *sql.DB) map[string][][]string {
	t.Helper()
	tables := []string{"posts", "comments", "like_events", "comment_like_events", "comment_attempts", "comment_mutes", "comment_challenge_uses", "admin_sessions"}
	contents := make(map[string][][]string, len(tables))
	for _, table := range tables {
		query := "SELECT * FROM " + table + " ORDER BY rowid"
		if table == "posts" {
			query = "SELECT id, title, blocks_json, likes_count, created_at, preview_media FROM posts ORDER BY id"
		}
		rows, err := db.Query(query)
		if err != nil {
			t.Fatalf("read %s: %v", table, err)
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(columns))
			destinations := make([]any, len(columns))
			for index := range values {
				destinations[index] = &values[index]
			}
			if err := rows.Scan(destinations...); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			record := make([]string, len(values))
			for index, value := range values {
				if value == nil {
					record[index] = "<NULL>"
				} else {
					record[index] = fmt.Sprint(value)
				}
			}
			contents[table] = append(contents[table], record)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return contents
}

func assertDatabaseIntegrity(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := integrityCheck(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal(errors.New("foreign key check failed"))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}
