package httpapi

import (
	"context"
	"encoding/json"
	"github.com/Levipanic/stereoblog-v2/backend/internal/admin"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
	"log/slog"
	"strings"
	"testing"
)

func TestModerationOnMigratedV1(t *testing.T) {
	_, _, cfg := adminTestRouter(t)
	path := testfixture.V1Database(t)
	db, schema, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := database.Migrate(context.Background(), db, path, schema, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE comment_mutes SET muted_until='2099-01-01 00:00:00' WHERE id=50; INSERT INTO comment_mutes (ip_hash,muted_until) VALUES ('expired','2000-01-01 00:00:00')"); err != nil {
		t.Fatal(err)
	}
	r, err := NewRouter(cfg, slog.Default(), db)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ method, path string }{{"GET", "/api/v1/admin/moderation"}, {"POST", "/api/v1/admin/comments/12/approve"}, {"POST", "/api/v1/admin/comments/12/reject"}, {"DELETE", "/api/v1/admin/comments/10"}, {"DELETE", "/api/v1/admin/comment-mutes/50"}} {
		if res := adminRequest(r, tc.method, tc.path, "", nil, ""); res.Code != 401 {
			t.Fatal("unprotected moderation")
		}
	}
	cookie, csrf := adminLogin(t, r)
	res := adminRequest(r, "GET", "/api/v1/admin/moderation", "", cookie, "")
	var overview admin.Overview
	if res.Code != 200 || json.Unmarshal(res.Body.Bytes(), &overview) != nil || len(overview.Pending) != 1 || overview.Pending[0].ID != 12 || len(overview.Attempts) != 3 || len(overview.Mutes) != 1 {
		t.Fatalf("overview: %d %s", res.Code, res.Body.String())
	}
	if strings.Contains(res.Body.String(), "fixture-muted-ip-hash") || strings.Contains(res.Body.String(), "fixture-attempt-ip-hash") {
		t.Fatal("full hash leaked")
	}
	if res := adminRequest(r, "POST", "/api/v1/admin/comments/12/approve", "", cookie, ""); res.Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	for _, action := range []string{"approve", "reject"} {
		res = adminRequest(r, "POST", "/api/v1/admin/comments/12/"+action, "", cookie, csrf)
		if res.Code != 200 {
			t.Fatal(res.Body.String())
		}
		res = performRequest(r, "GET", "/api/v1/posts/1/comments", "")
		if strings.Contains(res.Body.String(), "Pending comment") != (action == "approve") {
			t.Fatal("public visibility incorrect")
		}
	}
	if _, err := db.Exec("INSERT INTO comments (id,post_id,parent_id,content) VALUES (14,1,11,'grandchild'); INSERT INTO comment_like_events(comment_id,ip_hash) VALUES (14,'grandchild-like')"); err != nil {
		t.Fatal(err)
	}
	res = adminRequest(r, "DELETE", "/api/v1/admin/comments/10", "", cookie, csrf)
	if res.Code != 200 {
		t.Fatal(res.Body.String())
	}
	var n int
	if err := db.QueryRow("SELECT count(*) FROM comments WHERE id IN (10,11,14)").Scan(&n); err != nil || n != 0 {
		t.Fatal("subtree remains")
	}
	if err := db.QueryRow("SELECT count(*) FROM comment_like_events").Scan(&n); err != nil || n != 0 {
		t.Fatal("subtree likes remain")
	}
	if err := db.QueryRow("SELECT count(*) FROM comments WHERE id IN (12,13)").Scan(&n); err != nil || n != 2 {
		t.Fatal("unrelated comments deleted")
	}
	res = adminRequest(r, "DELETE", "/api/v1/admin/comment-mutes/50", "", cookie, csrf)
	if res.Code != 200 {
		t.Fatal(res.Body.String())
	}
	res = adminRequest(r, "GET", "/api/v1/admin/moderation", "", cookie, "")
	json.Unmarshal(res.Body.Bytes(), &overview)
	if len(overview.Mutes) != 0 {
		t.Fatal("unmute failed")
	}
	for _, path := range []string{"/api/v1/admin/comments/999/approve", "/api/v1/admin/comments/999/reject"} {
		if res := adminRequest(r, "POST", path, "", cookie, csrf); res.Code != 404 {
			t.Fatal(res.Code)
		}
	}
	if res := adminRequest(r, "DELETE", "/api/v1/admin/comments/no", "", cookie, csrf); res.Code != 400 {
		t.Fatal(res.Code)
	}
}
