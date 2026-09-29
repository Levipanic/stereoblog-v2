package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"github.com/Levipanic/stereoblog-v2/backend/internal/admin"
	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func adminTestRouter(t *testing.T) (*gin.Engine, *sql.DB, config.Config) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Admin.Secret = "test-secret"
	cfg.Admin.SessionHashSalt = "test-salt"
	cfg.Admin.LoginRateLimitMax = 100
	cfg.Storage.UploadsPath = t.TempDir()
	db := newTestDatabase(t)
	r, err := NewRouter(cfg, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), db)
	if err != nil {
		t.Fatal(err)
	}
	return r, db, cfg
}
func adminRequest(r http.Handler, method, path, body string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrf)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}
func adminLogin(t *testing.T, r http.Handler) (*http.Cookie, string) {
	t.Helper()
	res := adminRequest(r, "POST", "/api/v1/admin/login", `{"secret":"test-secret"}`, nil, "")
	if res.Code != 200 {
		t.Fatalf("login: %d %s", res.Code, res.Body.String())
	}
	var session admin.Session
	if err := json.Unmarshal(res.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}
	cookies := res.Result().Cookies()
	if len(cookies) != 1 || !session.Authenticated || session.CSRF == "" || strings.Contains(res.Body.String(), cookies[0].Value) {
		t.Fatal("invalid session response")
	}
	return cookies[0], session.CSRF
}
func TestAdminSessionLifecycle(t *testing.T) {
	r, db, _ := adminTestRouter(t)
	if res := adminRequest(r, "POST", "/api/v1/admin/logout", "", nil, ""); res.Code != 401 {
		t.Fatal(res.Code)
	}
	if res := adminRequest(r, "POST", "/api/v1/admin/login", `{"secret":"wrong"}`, nil, ""); res.Code != 401 {
		t.Fatal(res.Code)
	}
	cookie, csrf := adminLogin(t, r)
	if !cookie.HttpOnly || cookie.Secure || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie: %#v", cookie)
	}
	var hash string
	if err := db.QueryRow("SELECT token_hash FROM admin_sessions").Scan(&hash); err != nil || hash == cookie.Value || len(hash) != 64 {
		t.Fatalf("stored token: %q %v", hash, err)
	}
	res := adminRequest(r, "GET", "/api/v1/admin/session", "", cookie, "")
	if res.Code != 200 || !strings.Contains(res.Body.String(), csrf) || res.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(res.Body.String())
	}
	for _, token := range []string{"", "wrong"} {
		if res := adminRequest(r, "POST", "/api/v1/admin/logout", "", cookie, token); res.Code != 403 {
			t.Fatal(res.Code)
		}
	}
	other, otherCSRF := adminLogin(t, r)
	if res := adminRequest(r, "POST", "/api/v1/admin/logout", "", cookie, otherCSRF); res.Code != 403 {
		t.Fatal("CSRF not session-bound")
	}
	res = adminRequest(r, "POST", "/api/v1/admin/logout", "", cookie, csrf)
	if res.Code != 200 || res.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout failed")
	}
	if res := adminRequest(r, "POST", "/api/v1/admin/logout", "", cookie, csrf); res.Code != 401 {
		t.Fatal("session not revoked")
	}
	if _, err := db.Exec("UPDATE admin_sessions SET expires_at = '2000-01-01 00:00:00'"); err != nil {
		t.Fatal(err)
	}
	if res := adminRequest(r, "POST", "/api/v1/admin/logout", "", other, otherCSRF); res.Code != 401 {
		t.Fatal("expired session accepted")
	}
	cookie, csrf = adminLogin(t, r)
	if _, err := db.Exec("UPDATE admin_sessions SET created_at = '2099-01-01 00:00:00'"); err != nil {
		t.Fatal(err)
	}
	if res := adminRequest(r, "POST", "/api/v1/admin/logout", "", cookie, csrf); res.Code != 401 {
		t.Fatal("future session accepted")
	}
}
func TestAdminLoginBoundaries(t *testing.T) {
	_, db, cfg := adminTestRouter(t)
	cfg.Server.Environment = "production"
	cfg.Admin.LoginRateLimitMax = 2
	cfg.Admin.SessionTTL = time.Hour
	r, err := NewRouter(cfg, slog.Default(), db)
	if err != nil {
		t.Fatal(err)
	}
	cookie, _ := adminLogin(t, r)
	if !cookie.Secure {
		t.Fatal("production cookie not secure")
	}
	res := adminRequest(r, "POST", "/api/v1/admin/login", `null`, nil, "")
	if res.Code != 400 {
		t.Fatal(res.Code)
	}
	res = adminRequest(r, "POST", "/api/v1/admin/login", `{"secret":"test-secret"}`, nil, "")
	if res.Code != 429 || res.Header().Get("Retry-After") == "" {
		t.Fatal("login limiter failed")
	}
	r, _, _ = adminTestRouter(t)
	for _, body := range []string{`[]`, `{}`, `{"secret":123}`, `{"secret":"test-secret"} {}`, strings.Repeat("x", 300000)} {
		res := adminRequest(r, "POST", "/api/v1/admin/login", body, nil, "")
		if res.Code < 400 {
			t.Fatal("invalid body accepted")
		}
	}
	req := httptest.NewRequest("POST", "/api/v1/admin/login", strings.NewReader(`{"secret":"test-secret"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != 403 {
		t.Fatal("cross-site login accepted")
	}
	if res := performRequest(r, "POST", "/api/v1/admin/login", `{"secret":"test-secret"}`); res.Code != 415 {
		t.Fatal("content type not enforced")
	}
}
