package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
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

func TestRecoveryDoesNotExposePanic(t *testing.T) {
	var logs bytes.Buffer
	router, err := NewRouter(config.Config{}, slog.New(slog.NewTextHandler(&logs, nil)))
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
	router, err := NewRouter(config.Config{}, slog.New(slog.NewTextHandler(&logs, nil)))
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
	router, err := NewRouter(cfg, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return router
}

func performRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
