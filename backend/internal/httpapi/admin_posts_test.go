package httpapi

import (
	"encoding/json"
	"fmt"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
	"strings"
	"testing"
)

func TestAdminPostCRUD(t *testing.T) {
	r, db, _ := adminTestRouter(t)
	const body = `{"title":"Новый пост","blocks":[{"type":"paragraph","text":"old fallback","content":[{"type":"text","text":"Rich text","marks":[{"type":"bold"}]}]},{"type":"media","mediaKind":"audio","src":"/uploads/test.mp3","spoiler":true,"caption":"Track"}],"preview_media":{"mediaKind":"image","src":"/uploads/cover.jpg"}}`
	for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
		path := "/api/v1/admin/posts"
		if method == "PUT" || method == "DELETE" {
			path += "/1"
		}
		if res := adminRequest(r, method, path, body, nil, ""); res.Code != 401 {
			t.Fatalf("unprotected %s: %d", method, res.Code)
		}
	}
	cookie, csrf := adminLogin(t, r)
	if res := adminRequest(r, "POST", "/api/v1/admin/posts", body, cookie, ""); res.Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	res := adminRequest(r, "POST", "/api/v1/admin/posts", body, cookie, csrf)
	if res.Code != 201 {
		t.Fatalf("create: %d %s", res.Code, res.Body.String())
	}
	var p posts.AdminPost
	if err := json.Unmarshal(res.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Slug != "новый-пост" || strings.Contains(string(p.Blocks), "old fallback") || !strings.Contains(string(p.Blocks), `"text":"Rich text"`) {
		t.Fatalf("canonical: %#v", p)
	}
	path := fmt.Sprintf("/api/v1/admin/posts/%d", p.ID)
	res = adminRequest(r, "POST", "/api/v1/admin/posts", body, cookie, csrf)
	if res.Code != 201 || !strings.Contains(res.Body.String(), "новый-пост-2") {
		t.Fatal("automatic collision handling failed")
	}
	input := posts.WriteInput{Title: "Changed title", Blocks: p.Blocks, Slug: p.Slug}
	raw, _ := json.Marshal(input)
	// Omit preview entirely, rather than sending null, to preserve explicit selection.
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	delete(fields, "preview_media")
	raw, _ = json.Marshal(fields)
	res = adminRequest(r, "PUT", path, string(raw), cookie, csrf)
	if res.Code != 200 || !strings.Contains(res.Body.String(), "новый-пост") || !strings.Contains(res.Body.String(), "cover.jpg") || !strings.Contains(res.Body.String(), `"spoiler":true`) {
		t.Fatalf("update lost fields: %d %s", res.Code, res.Body.String())
	}
	res = adminRequest(r, "GET", path, "", cookie, "")
	if res.Code != 200 {
		t.Fatal(res.Code)
	}
	res = adminRequest(r, "GET", "/api/v1/admin/posts?limit=1", "", cookie, "")
	if res.Code != 200 || !strings.Contains(res.Body.String(), "next_cursor") {
		t.Fatal(res.Body.String())
	}
	res = performRequest(r, "GET", "/api/v1/posts/новый-пост", "")
	if res.Code != 200 || !strings.Contains(res.Body.String(), "Rich text") {
		t.Fatal(res.Body.String())
	}
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"title":"x","slug":"new-slug","blocks":[{"type":"paragraph","text":"x"}]}`, 409},
		{`{"title":"x","blocks":[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","href":"javascript:alert(1)"}]}]}]}`, 400},
		{`{"title":"x","blocks":[{"type":"future-block"}]}`, 400},
		{`{"title":"x","blocks":[{"type":"paragraph","text":"x"}],"preview_media":{"mediaKind":"image","src":"https://evil/x"}}`, 400},
	} {
		res = adminRequest(r, "PUT", path, tc.body, cookie, csrf)
		if res.Code != tc.status {
			t.Fatalf("validation: %d %s", res.Code, res.Body.String())
		}
	}
	res = adminRequest(r, "POST", "/api/v1/admin/posts", `{"title":"x","slug":"новый-пост","blocks":[{"type":"paragraph","text":"x"}]}`, cookie, csrf)
	if res.Code != 409 {
		t.Fatal("explicit collision accepted")
	}
	if _, err := db.Exec("INSERT INTO comments (id,post_id,content) VALUES (10,?,'root'), (11,?,'reply')", p.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE comments SET parent_id=10 WHERE id=11; INSERT INTO comment_like_events (comment_id,ip_hash) VALUES (11,'test'); INSERT INTO like_events (post_id,ip_hash) VALUES (1,'test')"); err != nil {
		t.Fatal(err)
	}
	res = adminRequest(r, "DELETE", path, "", cookie, csrf)
	if res.Code != 200 {
		t.Fatal(res.Body.String())
	}
	for _, table := range []string{"comments", "like_events", "comment_like_events"} {
		var n int
		if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("cascade %s: %d %v", table, n, err)
		}
	}
	if res := adminRequest(r, "GET", path, "", cookie, ""); res.Code != 404 {
		t.Fatal(res.Code)
	}
}
