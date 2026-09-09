package testfixture

import (
	"database/sql"
	_ "embed"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

//go:embed testdata/v1.sql
var v1SQL string

func V1Database(t testing.TB) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "blog.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(v1SQL); err != nil {
		db.Close()
		t.Fatalf("create v1 fixture: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close v1 fixture: %v", err)
	}
	return path
}
