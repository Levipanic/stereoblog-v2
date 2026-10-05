package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Sanitized schema from the July 2026 archive's backend/db.js.
const earlyFixture = `
CREATE TABLE posts (id INTEGER PRIMARY KEY AUTOINCREMENT, title TEXT NOT NULL, blocks_json TEXT NOT NULL, likes_count INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT (datetime('now')));
CREATE TABLE comments (id INTEGER PRIMARY KEY AUTOINCREMENT, post_id INTEGER NOT NULL, parent_id INTEGER, name TEXT, content TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (datetime('now')), FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE, FOREIGN KEY(parent_id) REFERENCES comments(id) ON DELETE CASCADE);
CREATE TABLE like_events (id INTEGER PRIMARY KEY AUTOINCREMENT, post_id INTEGER NOT NULL, ip_hash TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (datetime('now')), FOREIGN KEY(post_id) REFERENCES posts(id) ON DELETE CASCADE);
CREATE TABLE admin_sessions (id INTEGER PRIMARY KEY AUTOINCREMENT, token_hash TEXT NOT NULL UNIQUE, created_at TEXT NOT NULL DEFAULT (datetime('now')), expires_at TEXT NOT NULL);
CREATE INDEX idx_comments_post_parent ON comments(post_id,parent_id,id);
CREATE INDEX idx_like_events_post_ip_created ON like_events(post_id,ip_hash,created_at);
CREATE INDEX idx_admin_sessions_expires_at ON admin_sessions(expires_at);
INSERT INTO posts VALUES (7,'Старый пост','[{"type":"paragraph","text":"Original"}]',85,'2026-07-07 12:00:00');
INSERT INTO comments VALUES (10,7,NULL,NULL,'Root','2026-07-07 12:01:00'),(11,7,10,'Reader','Reply','2026-07-07 12:02:00');
INSERT INTO like_events VALUES (1,7,'synthetic-hash','2026-07-07 12:03:00');
INSERT INTO admin_sessions VALUES (1,'synthetic-token','2026-07-07 12:00:00','2026-07-08 12:00:00');`

func TestEarlyMigration(t *testing.T) {
	ctx := context.Background()
	for _, corrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "preserves data and backup", true: "rejects partial schema"}[corrupt], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "blog.db")
			raw, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := raw.Exec(earlyFixture); err != nil {
				t.Fatal(err)
			}
			if corrupt {
				if _, err := raw.Exec("ALTER TABLE comments ADD COLUMN status TEXT"); err != nil {
					t.Fatal(err)
				}
			}
			raw.Close()
			db, schema, err := Open(ctx, path)
			if corrupt {
				if err == nil {
					db.Close()
					t.Fatal("partial schema accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if schema != SchemaEarlyV1 {
				t.Fatal(schema)
			}
			result, err := Migrate(ctx, db, path, schema, true)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(result.BackupPath); err != nil {
				t.Fatal(err)
			}
			backup, backupSchema, err := Open(ctx, result.BackupPath)
			if err != nil {
				t.Fatal(err)
			}
			defer backup.Close()
			if backupSchema != SchemaEarlyV1 {
				t.Fatal(backupSchema)
			}
			queries := []string{
				"SELECT id,title,blocks_json,likes_count,created_at FROM posts ORDER BY id",
				"SELECT id,post_id,parent_id,name,content,created_at FROM comments ORDER BY id",
				"SELECT * FROM like_events ORDER BY id", "SELECT * FROM admin_sessions ORDER BY id",
			}
			for _, query := range queries {
				read := func(db *sql.DB) [][]any {
					rows, err := db.Query(query)
					if err != nil {
						t.Fatal(err)
					}
					defer rows.Close()
					cols, _ := rows.Columns()
					var result [][]any
					for rows.Next() {
						values := make([]any, len(cols))
						pointers := make([]any, len(cols))
						for i := range values {
							pointers[i] = &values[i]
						}
						if err := rows.Scan(pointers...); err != nil {
							t.Fatal(err)
						}
						result = append(result, values)
					}
					if err := rows.Err(); err != nil {
						t.Fatal(err)
					}
					return result
				}
				if !reflect.DeepEqual(read(db), read(backup)) {
					t.Fatal("changed original data:", query)
				}
			}
			assertDatabaseIntegrity(t, db)
			if result, err := Migrate(ctx, db, path, SchemaV1, true); err != nil || result.Applied != 0 {
				t.Fatalf("repeat: %+v %v", result, err)
			}
		})
	}
}
