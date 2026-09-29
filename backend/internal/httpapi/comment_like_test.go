package httpapi

import (
	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCommentLikeHTTP(t *testing.T) {
	r := newFixtureRouterWithConfig(t, config.Config{Likes: config.Likes{Cooldown: time.Minute, RateLimitMax: 5, IPHashSalt: "salt"}})
	for _, tc := range []struct {
		path   string
		status int
	}{
		{"/api/v1/comments/12/likes", 404}, {"/api/v1/comments/no/likes", 400},
		{"/api/v1/comments/10/likes", 200}, {"/api/v1/comments/10/likes", 429},
		{"/api/v1/posts/1/likes", 200}, {"/api/v1/comments/11/likes", 429},
	} {
		res := performRequest(r, "POST", tc.path, "")
		if res.Code != tc.status || strings.Contains(res.Body.String(), "ip_hash") {
			t.Fatalf("%s: %d %s", tc.path, res.Code, res.Body.String())
		}
		if res.Code == 429 && res.Header().Get("Retry-After") == "" {
			t.Fatal("missing retry header")
		}
	}
	req := httptest.NewRequest("POST", "/api/v1/comments/10/likes", nil)
	req.Header.Set("Origin", "https://other.example")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != 403 {
		t.Fatalf("cross origin: %d", res.Code)
	}
}
