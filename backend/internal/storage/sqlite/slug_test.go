package sqlite

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
)

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"Тестовая публикация": "тестовая-публикация",
		"  Hello, WORLD!  ":   "hello-world",
		"Ёж & Café":           "ёж-café",
		"Cafe\u0301":          "cafe\u0301",
		"one---two___three":   "one-two-three",
		"123 + тест":          "123-тест",
		"!!!":                 "post",
		"\u0301 --":           "post",
	}
	for title, want := range tests {
		if got := Slugify(title); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestSlugMigrationBackfillsCollisionsWithoutChangingLegacyData(t *testing.T) {
	path := testfixture.V1Database(t)
	db, schema, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`INSERT INTO posts (id, title, blocks_json, created_at) VALUES
		(3, 'Same', '[]', '2024-01-03 00:00:00'),
		(4, 'Same 5', '[]', '2024-01-04 00:00:00'),
		(5, 'Same', '[]', '2024-01-05 00:00:00'),
		(6, '!!!', '[]', '2024-01-06 00:00:00'),
		(7, '???', '[]', '2024-01-07 00:00:00')`); err != nil {
		t.Fatal(err)
	}
	before := databaseContents(t, db)

	result, err := Migrate(context.Background(), db, path, schema, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != 2 || result.Applied != 2 {
		t.Fatalf("unexpected migration result: %#v", result)
	}
	want := []string{"тестовая-публикация", "english-fixture-post", "same", "same-5", "same-5-2", "post", "post-7"}
	if got := postSlugs(t, db); !reflect.DeepEqual(got, want) {
		t.Fatalf("slugs = %q, want %q", got, want)
	}
	if after := databaseContents(t, db); !reflect.DeepEqual(after, before) {
		t.Fatal("slug migration changed legacy post or related data")
	}

	result, err = Migrate(context.Background(), db, path, SchemaV1, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied != 0 || !reflect.DeepEqual(postSlugs(t, db), want) {
		t.Fatal("repeated migration changed stable slugs")
	}
}

func TestProductionUpgradeFromMigrationOneBacksUpBeforeSlugs(t *testing.T) {
	path := testfixture.V1Database(t)
	db, schema, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if applied, err := applyMigrations(context.Background(), db, migrations[:1]); err != nil || applied != 1 {
		t.Fatalf("prepare version 1: applied=%d error=%v", applied, err)
	}

	result, err := Migrate(context.Background(), db, path, schema, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != 2 || result.Applied != 1 || result.BackupPath == "" {
		t.Fatalf("unexpected production upgrade result: %#v", result)
	}
	backup, err := sql.Open("sqlite", result.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()
	var slugColumn int
	if err := backup.QueryRow("SELECT count(*) FROM pragma_table_info('posts') WHERE name = 'slug'").Scan(&slugColumn); err != nil {
		t.Fatal(err)
	}
	if slugColumn != 0 {
		t.Fatal("pre-slug backup contains the slug migration")
	}
}

func TestCompletedSlugMigrationIsValidatedOnStartup(t *testing.T) {
	tests := map[string]string{
		"missing index": "DROP INDEX idx_posts_slug",
		"wrong index":   "DROP INDEX idx_posts_slug; CREATE UNIQUE INDEX idx_posts_slug ON posts(title)",
		"partial index": "DROP INDEX idx_posts_slug; CREATE UNIQUE INDEX idx_posts_slug ON posts(slug) WHERE slug IS NOT NULL",
		"null slug":     "UPDATE posts SET slug = NULL WHERE id = 1",
		"empty slug":    "UPDATE posts SET slug = '' WHERE id = 1",
	}
	for name, corrupt := range tests {
		t.Run(name, func(t *testing.T) {
			path := testfixture.V1Database(t)
			db, schema, err := Open(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := Migrate(context.Background(), db, path, schema, false); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(corrupt); err != nil {
				t.Fatal(err)
			}
			if _, err := Migrate(context.Background(), db, path, SchemaV1, false); err == nil {
				t.Fatal("invalid completed slug migration was accepted")
			}
		})
	}
}

func TestCompletedSlugMigrationAllowsFutureAppendedColumns(t *testing.T) {
	path := testfixture.V1Database(t)
	db, schema, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := Migrate(context.Background(), db, path, schema, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("ALTER TABLE posts ADD COLUMN future_value TEXT"); err != nil {
		t.Fatal(err)
	}
	if result, err := Migrate(context.Background(), db, path, SchemaV1, false); err != nil || result.Applied != 0 {
		t.Fatalf("forward-compatible column rejected: result=%#v error=%v", result, err)
	}
}

func TestMalformedSlugColumnAndFailedPostconditionAreRejected(t *testing.T) {
	t.Run("malformed completed column", func(t *testing.T) {
		path := testfixture.V1Database(t)
		db, _, err := Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		if applied, err := applyMigrations(context.Background(), db, migrations[:1]); err != nil || applied != 1 {
			t.Fatalf("prepare version 1: applied=%d error=%v", applied, err)
		}
		if _, err := db.Exec("ALTER TABLE posts ADD COLUMN slug INTEGER; UPDATE posts SET slug = id; CREATE UNIQUE INDEX idx_posts_slug ON posts(slug); INSERT INTO schema_migrations (version, name) VALUES (2, 'post_slugs')"); err != nil {
			t.Fatal(err)
		}
		if _, err := Migrate(context.Background(), db, path, SchemaV1, false); err == nil {
			t.Fatal("malformed completed slug column was accepted")
		}
	})

	t.Run("postcondition rollback", func(t *testing.T) {
		path := testfixture.V1Database(t)
		db, _, err := Open(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		bad := []migration{
			migrations[0],
			{version: 2, name: "post_slugs", apply: func(ctx context.Context, conn *sql.Conn) error {
				_, err := conn.ExecContext(ctx, "ALTER TABLE posts ADD COLUMN slug TEXT; UPDATE posts SET slug = id")
				return err
			}},
		}
		if _, err := applyMigrations(context.Background(), db, bad); err == nil {
			t.Fatal("migration without slug index passed postcondition validation")
		}
		var slugColumn, metadata int
		if err := db.QueryRow("SELECT count(*) FROM pragma_table_info('posts') WHERE name = 'slug'").Scan(&slugColumn); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name = 'schema_migrations'").Scan(&metadata); err != nil {
			t.Fatal(err)
		}
		if slugColumn != 0 || metadata != 0 {
			t.Fatalf("failed migration leaked schema: slug=%d metadata=%d", slugColumn, metadata)
		}
	})
}

func postSlugs(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query("SELECT slug FROM posts ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			t.Fatal(err)
		}
		slugs = append(slugs, slug)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return slugs
}
