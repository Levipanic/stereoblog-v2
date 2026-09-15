package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
)

func TestHealthAndSecurityHeaders(t *testing.T) {
	router := newTestRouter(t, config.Config{})
	response := performRequest(router, http.MethodGet, "/api/v1/health", "")

	if response.Code != http.StatusOK || response.Body.String() != "{\"status\":\"ok\"}" {
		t.Fatalf("unexpected health response: %d %s", response.Code, response.Body.String())
	}
	for name, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "same-origin",
	} {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestRouterRequiresDatabase(t *testing.T) {
	if router, err := NewRouter(config.Config{}, slog.Default(), nil); err == nil || router != nil {
		t.Fatalf("nil database accepted: router=%v error=%v", router, err)
	}
}

func TestJSONErrors(t *testing.T) {
	router := newTestRouter(t, config.Config{})
	tests := []struct {
		method string
		path   string
		status int
		code   string
	}{
		{http.MethodGet, "/api/v1/missing", http.StatusNotFound, "not_found"},
		{http.MethodPost, "/api/v1/health", http.StatusMethodNotAllowed, "method_not_allowed"},
	}

	for _, tt := range tests {
		response := performRequest(router, tt.method, tt.path, "")
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON error: %v", err)
		}
		if response.Code != tt.status || body.Error.Code != tt.code || body.Error.Message == "" {
			t.Fatalf("unexpected error response: %d %#v", response.Code, body)
		}
	}
}

func TestFeedAPIContractAndValidation(t *testing.T) {
	path := testfixture.V1Database(t)
	db, schema, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(context.Background(), db, path, schema, false); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router, err := NewRouter(config.Config{}, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), db)
	if err != nil {
		t.Fatal(err)
	}

	response := performRequest(router, http.MethodGet, "/api/v1/posts?limit=1", "")
	var first posts.FeedPage
	if err := json.Unmarshal(response.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(first.Items) != 1 || first.Items[0].ID != 2 || first.Items[0].CommentCount != 0 || first.NextCursor == nil {
		t.Fatalf("unexpected first feed page: %d %#v", response.Code, first)
	}
	if strings.Contains(response.Body.String(), "blocks_json") || strings.Contains(response.Body.String(), "future-block") {
		t.Fatalf("feed leaked full post content: %s", response.Body.String())
	}

	response = performRequest(router, http.MethodGet, "/api/v1/posts?limit=1&cursor="+url.QueryEscape(*first.NextCursor), "")
	var second posts.FeedPage
	if err := json.Unmarshal(response.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(second.Items) != 1 || second.Items[0].ID != 1 || second.NextCursor != nil || second.Items[0].CommentCount != 2 {
		t.Fatalf("unexpected second feed page: %d %#v", response.Code, second)
	}

	for requestPath, code := range map[string]string{
		"/api/v1/posts?limit=0":       "invalid_limit",
		"/api/v1/posts?limit=51":      "invalid_limit",
		"/api/v1/posts?limit=text":    "invalid_limit",
		"/api/v1/posts?cursor=broken": "invalid_cursor",
	} {
		response = performRequest(router, http.MethodGet, requestPath, "")
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusBadRequest || body.Error.Code != code {
			t.Fatalf("%s returned %d %#v", requestPath, response.Code, body)
		}
	}
}

func TestRecoveryDoesNotExposePanic(t *testing.T) {
	var logs bytes.Buffer
	router, err := NewRouter(config.Config{}, slog.New(slog.NewTextHandler(&logs, nil)), newTestDatabase(t))
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/test/panic", func(*gin.Context) { panic("private panic detail") })

	response := performRequest(router, http.MethodGet, "/test/panic", "")
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "private") {
		t.Fatalf("unsafe recovery response: %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(logs.String(), "panic recovered") {
		t.Fatalf("panic was not logged: %s", logs.String())
	}
}

func TestRecoveryDoesNotAppendJSONAfterResponseStarted(t *testing.T) {
	router := newTestRouter(t, config.Config{})
	router.GET("/test/partial-panic", func(c *gin.Context) {
		c.String(http.StatusOK, "partial")
		panic("private panic detail")
	})

	response := performRequest(router, http.MethodGet, "/test/partial-panic", "")
	if response.Code != http.StatusOK || response.Body.String() != "partial" {
		t.Fatalf("recovery corrupted started response: %d %q", response.Code, response.Body.String())
	}
}

func TestRequestLogOmitsQueryAndHeaders(t *testing.T) {
	var logs bytes.Buffer
	router, err := NewRouter(config.Config{}, slog.New(slog.NewTextHandler(&logs, nil)), newTestDatabase(t))
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health?token=query-secret", nil)
	request.Header.Set("Authorization", "header-secret")
	router.ServeHTTP(httptest.NewRecorder(), request)
	if strings.Contains(logs.String(), "query-secret") || strings.Contains(logs.String(), "header-secret") {
		t.Fatalf("request log leaked sensitive request data: %s", logs.String())
	}
}

func TestClientIPTrustsOnlyConfiguredProxy(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want string
	}{
		{"untrusted", config.Config{}, "10.0.0.2"},
		{"trusted", config.Config{Server: config.Server{TrustProxy: true, TrustedProxyCIDR: []string{"10.0.0.0/8"}}}, "203.0.113.9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestRouter(t, tt.cfg)
			router.GET("/test/ip", func(c *gin.Context) { c.String(http.StatusOK, ClientIP(c)) })
			request := httptest.NewRequest(http.MethodGet, "/test/ip", nil)
			request.RemoteAddr = "10.0.0.2:1234"
			request.Header.Set("X-Forwarded-For", "203.0.113.9")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Body.String() != tt.want {
				t.Fatalf("client IP = %q, want %q", response.Body.String(), tt.want)
			}
		})
	}
}

func newTestRouter(t *testing.T, cfg config.Config) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router, err := NewRouter(cfg, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), newTestDatabase(t))
	if err != nil {
		t.Fatal(err)
	}
	return router
}

func newTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	path := t.TempDir() + "/blog.db"
	db, schema, err := database.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(context.Background(), db, path, schema, false); err != nil {
		t.Fatal(err)
	}
	return db
}

func performRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
