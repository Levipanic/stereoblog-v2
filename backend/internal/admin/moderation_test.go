package admin

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
)

func TestDeleteSubtreeWithoutLegacyParentForeignKey(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA foreign_keys=ON;
	CREATE TABLE comments(id INTEGER PRIMARY KEY,post_id INTEGER,parent_id INTEGER);
	CREATE TABLE comment_like_events(comment_id INTEGER REFERENCES comments(id) ON DELETE CASCADE);
	INSERT INTO comments VALUES (1,10,NULL),(2,10,1),(3,10,2),(4,10,NULL);
	INSERT INTO comment_like_events VALUES (3);`); err != nil {
		t.Fatal(err)
	}
	postID, err := NewModeration(db).DeleteComment(context.Background(), 1)
	if err != nil || postID != 10 {
		t.Fatalf("delete: %d %v", postID, err)
	}
	var comments, likes int
	if err := db.QueryRow("SELECT count(*) FROM comments").Scan(&comments); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM comment_like_events").Scan(&likes); err != nil {
		t.Fatal(err)
	}
	if comments != 1 || likes != 0 {
		t.Fatalf("subtree remains: comments=%d likes=%d", comments, likes)
	}
}
