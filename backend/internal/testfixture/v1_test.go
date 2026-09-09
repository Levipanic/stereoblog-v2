package testfixture

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestV1DatabaseSchemaAndIntegrity(t *testing.T) {
	db := openFixture(t, V1Database(t))

	wantColumns := map[string]string{
		"posts":                  "id:INTEGER:0:-:1,title:TEXT:1:-:0,blocks_json:TEXT:1:-:0,likes_count:INTEGER:1:0:0,created_at:TEXT:1:datetime('now'):0,preview_media:TEXT:0:NULL:0",
		"comments":               "id:INTEGER:0:-:1,post_id:INTEGER:1:-:0,parent_id:INTEGER:0:-:0,name:TEXT:0:-:0,content:TEXT:1:-:0,created_at:TEXT:1:datetime('now'):0,likes_count:INTEGER:1:0:0,status:TEXT:1:'visible':0,moderation_reason:TEXT:0:-:0,text_hash:TEXT:0:-:0,text_fingerprint:TEXT:0:-:0",
		"like_events":            "id:INTEGER:0:-:1,post_id:INTEGER:1:-:0,ip_hash:TEXT:1:-:0,created_at:TEXT:1:datetime('now'):0",
		"comment_like_events":    "id:INTEGER:0:-:1,comment_id:INTEGER:1:-:0,ip_hash:TEXT:1:-:0,created_at:TEXT:1:datetime('now'):0",
		"comment_attempts":       "id:INTEGER:0:-:1,ip_hash:TEXT:1:-:0,post_id:INTEGER:0:-:0,status:TEXT:1:-:0,reason:TEXT:0:-:0,content:TEXT:0:-:0,text_hash:TEXT:0:-:0,fingerprint:TEXT:0:-:0,created_at:TEXT:1:datetime('now'):0",
		"comment_mutes":          "id:INTEGER:0:-:1,ip_hash:TEXT:1:-:0,reason:TEXT:0:-:0,muted_until:TEXT:1:-:0,mute_count:INTEGER:1:1:0,created_at:TEXT:1:datetime('now'):0",
		"comment_challenge_uses": "token_hash:TEXT:0:-:1,post_id:INTEGER:1:-:0,used_count:INTEGER:1:0:0,first_used_at:TEXT:1:datetime('now'):0,last_used_at:TEXT:1:datetime('now'):0",
		"admin_sessions":         "id:INTEGER:0:-:1,token_hash:TEXT:1:-:0,created_at:TEXT:1:datetime('now'):0,expires_at:TEXT:1:-:0",
	}

	if got := objectNames(t, db, "table"); !reflect.DeepEqual(got, sortedKeys(wantColumns)) {
		t.Fatalf("tables = %v, want %v", got, sortedKeys(wantColumns))
	}
	for table, want := range wantColumns {
		if got := columnSignature(t, db, table); got != want {
			t.Errorf("%s columns = %v, want %v", table, got, want)
		}
	}

	wantIndexes := map[string]string{
		"idx_admin_sessions_expires_at":              "expires_at",
		"idx_comment_attempts_ip_post_hash_time":     "ip_hash,post_id,text_hash,created_at",
		"idx_comment_attempts_ip_status_time":        "ip_hash,status,created_at",
		"idx_comment_attempts_ip_time":               "ip_hash,created_at",
		"idx_comment_attempts_post_time":             "post_id,created_at",
		"idx_comment_like_events_comment_ip_created": "comment_id,ip_hash,created_at",
		"idx_comment_mutes_until":                    "muted_until",
		"idx_comments_post_parent":                   "post_id,parent_id,id",
		"idx_comments_status_created":                "status,created_at",
		"idx_like_events_post_ip_created":            "post_id,ip_hash,created_at",
	}
	if got := objectNames(t, db, "index"); !reflect.DeepEqual(got, sortedKeys(wantIndexes)) {
		t.Fatalf("indexes = %v, want %v", got, sortedKeys(wantIndexes))
	}
	for index, want := range wantIndexes {
		if got := indexColumns(t, db, index); got != want {
			t.Errorf("%s columns = %s, want %s", index, got, want)
		}
	}

	wantUnique := map[string][]string{
		"admin_sessions":         {"token_hash"},
		"comment_challenge_uses": {"token_hash"},
		"comment_mutes":          {"ip_hash"},
	}
	for table, want := range wantUnique {
		if got := uniqueColumns(t, db, table); !reflect.DeepEqual(got, want) {
			t.Errorf("%s unique columns = %v, want %v", table, got, want)
		}
	}

	for _, table := range []string{"posts", "comments", "like_events", "comment_like_events", "comment_attempts", "comment_mutes", "admin_sessions"} {
		var ddl string
		if err := db.QueryRow("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&ddl); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(strings.ToUpper(ddl), "AUTOINCREMENT") {
			t.Errorf("%s is missing AUTOINCREMENT", table)
		}
	}

	wantForeignKeys := []string{
		"comment_like_events.comment_id->comments.id CASCADE",
		"comments.parent_id->comments.id CASCADE",
		"comments.post_id->posts.id CASCADE",
		"like_events.post_id->posts.id CASCADE",
	}
	var gotForeignKeys []string
	for _, table := range []string{"comments", "like_events", "comment_like_events"} {
		gotForeignKeys = append(gotForeignKeys, foreignKeys(t, db, table)...)
	}
	slices.Sort(gotForeignKeys)
	if !reflect.DeepEqual(gotForeignKeys, wantForeignKeys) {
		t.Fatalf("foreign keys = %v, want %v", gotForeignKeys, wantForeignKeys)
	}

	var integrity string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil || integrity != "ok" {
		t.Fatalf("integrity_check = %q, %v", integrity, err)
	}
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("fixture has a foreign-key violation")
	}
}

func TestV1DatabaseContentCoverage(t *testing.T) {
	db := openFixture(t, V1Database(t))
	wantCounts := map[string]int{
		"posts": 2, "comments": 4, "like_events": 1, "comment_like_events": 1,
		"comment_attempts": 3, "comment_mutes": 1, "comment_challenge_uses": 1, "admin_sessions": 1,
	}
	for table, want := range wantCounts {
		var got int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&got); err != nil || got != want {
			t.Errorf("%s count = %d, %v; want %d", table, got, err, want)
		}
	}

	rows, err := db.Query("SELECT blocks_json FROM posts ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	blockTypes := map[string]bool{}
	mediaKinds := map[string]bool{}
	spoiler := false
	unknown := false
	invalidHeading := false
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var blocks []map[string]any
		if err := json.Unmarshal([]byte(raw), &blocks); err != nil {
			t.Fatal(err)
		}
		for _, block := range blocks {
			kind, _ := block["type"].(string)
			blockTypes[kind] = true
			if kind == "media" {
				mediaKind, _ := block["mediaKind"].(string)
				mediaKinds[mediaKind] = true
				spoiler = spoiler || block["spoiler"] == true
			}
			unknown = unknown || kind == "future-block"
			invalidHeading = invalidHeading || kind == "heading" && block["level"] == float64(9)
		}
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"paragraph", "heading", "quote", "divider", "media"} {
		if !blockTypes[kind] {
			t.Errorf("missing block type %s", kind)
		}
	}
	for _, kind := range []string{"image", "gif", "video", "audio", "file"} {
		if !mediaKinds[kind] {
			t.Errorf("missing media kind %s", kind)
		}
	}
	if !spoiler || !unknown || !invalidHeading {
		t.Fatalf("edge coverage: spoiler=%v unknown=%v invalidHeading=%v", spoiler, unknown, invalidHeading)
	}

	var explicitPreview, nullPreview, nested, pending, rejected int
	queries := []struct {
		query string
		value *int
	}{
		{"SELECT count(*) FROM posts WHERE preview_media IS NOT NULL", &explicitPreview},
		{"SELECT count(*) FROM posts WHERE preview_media IS NULL", &nullPreview},
		{"SELECT count(*) FROM comments WHERE parent_id IS NOT NULL", &nested},
		{"SELECT count(*) FROM comments WHERE status = 'pending'", &pending},
		{"SELECT count(*) FROM comments WHERE status = 'rejected'", &rejected},
	}
	for _, item := range queries {
		if err := db.QueryRow(item.query).Scan(item.value); err != nil {
			t.Fatal(err)
		}
	}
	if explicitPreview != 1 || nullPreview != 1 || nested != 1 || pending != 1 || rejected != 1 {
		t.Fatalf("missing edge rows: preview=%d null=%d nested=%d pending=%d rejected=%d", explicitPreview, nullPreview, nested, pending, rejected)
	}

	var semanticRows int
	if err := db.QueryRow(`
		SELECT count(*) FROM posts
		WHERE (id = 1 AND title = 'Тестовая публикация' AND likes_count = 7 AND created_at = '2024-01-02 03:04:05')
		   OR (id = 2 AND title = 'English fixture post' AND likes_count = 0 AND created_at = '2024-01-02 03:04:05')
	`).Scan(&semanticRows); err != nil || semanticRows != 2 {
		t.Fatalf("post semantics = %d, %v", semanticRows, err)
	}
	if err := db.QueryRow(`
		SELECT count(*) FROM comments
		WHERE (id = 10 AND name IS NULL AND likes_count = 2)
		   OR (id = 12 AND name = '' AND status = 'pending')
	`).Scan(&semanticRows); err != nil || semanticRows != 2 {
		t.Fatalf("comment semantics = %d, %v", semanticRows, err)
	}
	if err := db.QueryRow(`
		SELECT
		  (SELECT count(*) FROM like_events WHERE post_id = 1) +
		  (SELECT count(*) FROM comment_like_events WHERE comment_id = 10) +
		  (SELECT count(*) FROM comment_attempts WHERE status IN ('visible', 'pending', 'muted')) +
		  (SELECT count(*) FROM comment_mutes WHERE mute_count = 2) +
		  (SELECT count(*) FROM comment_challenge_uses WHERE used_count = 2) +
		  (SELECT count(*) FROM admin_sessions WHERE token_hash = 'fixture-session-hash')
	`).Scan(&semanticRows); err != nil || semanticRows != 8 {
		t.Fatalf("state-table semantics = %d, %v", semanticRows, err)
	}
}

func TestV1DatabasesAreIsolated(t *testing.T) {
	firstPath := V1Database(t)
	secondPath := V1Database(t)
	if firstPath == secondPath {
		t.Fatal("fixtures share a path")
	}

	first := openFixture(t, firstPath)
	second := openFixture(t, secondPath)
	if _, err := first.Exec("DELETE FROM posts WHERE id = 1"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := second.QueryRow("SELECT count(*) FROM posts").Scan(&count); err != nil || count != 2 {
		t.Fatalf("second fixture changed: count=%d, err=%v", count, err)
	}
}

func openFixture(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		db.Close()
		t.Fatalf("foreign_keys = %d, %v", foreignKeys, err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func objectNames(t *testing.T, db *sql.DB, objectType string) []string {
	t.Helper()
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type = ? AND name NOT LIKE 'sqlite_%' ORDER BY name", objectType)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	return names
}

func columnSignature(t *testing.T, db *sql.DB, table string) string {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var id, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&id, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		value := "-"
		if defaultValue.Valid {
			value = defaultValue.String
		}
		columns = append(columns, fmt.Sprintf("%s:%s:%d:%s:%d", name, columnType, notNull, value, primaryKey))
	}
	return strings.Join(columns, ",")
}

func indexColumns(t *testing.T, db *sql.DB, index string) string {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf("PRAGMA index_info(%q)", index))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var sequence, columnID int
		var name string
		if err := rows.Scan(&sequence, &columnID, &name); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, name)
	}
	return strings.Join(columns, ",")
}

func uniqueColumns(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf("PRAGMA index_list(%q)", table))
	if err != nil {
		t.Fatal(err)
	}
	var indexes []string
	for rows.Next() {
		var sequence, unique, partial int
		var name, origin string
		if err := rows.Scan(&sequence, &name, &unique, &origin, &partial); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if unique == 1 {
			indexes = append(indexes, name)
		}
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	var columns []string
	for _, index := range indexes {
		columns = append(columns, indexColumns(t, db, index))
	}
	slices.Sort(columns)
	return columns
}

func foreignKeys(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%q)", table))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var id, sequence int
		var targetTable, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &sequence, &targetTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatal(err)
		}
		keys = append(keys, fmt.Sprintf("%s.%s->%s.%s %s", table, from, targetTable, to, onDelete))
	}
	return keys
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
