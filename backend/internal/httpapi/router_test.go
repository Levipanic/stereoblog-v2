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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/antispam"
	"github.com/Levipanic/stereoblog-v2/backend/internal/comments"
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
	router := newFixtureRouter(t)

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

func TestPostBySlugAndIDResolutionAPI(t *testing.T) {
	router := newFixtureRouter(t)
	response := performRequest(router, http.MethodGet, "/api/v1/posts/"+url.PathEscape("тестовая-публикация"), "")
	var post posts.Post
	if err := json.Unmarshal(response.Body.Bytes(), &post); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || post.ID != 1 || post.Slug != "тестовая-публикация" || post.Title != "Тестовая публикация" ||
		post.CreatedAt != "2024-01-02T03:04:05Z" || post.Likes != 7 || post.PreviewText != "Привет из синтетической фикстуры." ||
		post.PreviewMedia == nil || post.PreviewMedia.Src != "/uploads/fixture-image.jpg" || len(post.Blocks) != 9 || post.ReadingMinutes != 1 {
		t.Fatalf("unexpected post response: %d %#v", response.Code, post)
	}

	response = performRequest(router, http.MethodGet, "/api/v1/posts/english-fixture-post", "")
	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "future-block") || strings.Contains(response.Body.String(), "payload") || !strings.Contains(response.Body.String(), `"type":"unknown"`) {
		t.Fatalf("unsafe unknown block response: %d %s", response.Code, response.Body.String())
	}

	response = performRequest(router, http.MethodGet, "/api/v1/posts/by-id/1", "")
	var resolved posts.IDResolution
	if err := json.Unmarshal(response.Body.Bytes(), &resolved); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || resolved.ID != 1 || resolved.Slug != "тестовая-публикация" {
		t.Fatalf("unexpected ID resolution: %d %#v", response.Code, resolved)
	}

	for requestPath, want := range map[string]struct {
		status int
		code   string
	}{
		"/api/v1/posts/missing":    {http.StatusNotFound, "post_not_found"},
		"/api/v1/posts/by-id":      {http.StatusNotFound, "post_not_found"},
		"/api/v1/posts/by-id/999":  {http.StatusNotFound, "post_not_found"},
		"/api/v1/posts/by-id/nope": {http.StatusBadRequest, "invalid_post_id"},
	} {
		response = performRequest(router, http.MethodGet, requestPath, "")
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != want.status || body.Error.Code != want.code {
			t.Fatalf("%s returned %d %#v", requestPath, response.Code, body)
		}
	}
}

func TestPostLikeAPIEnforcesCooldownAndRateLimit(t *testing.T) {
	cfg := config.Config{Likes: config.Likes{Cooldown: time.Minute, RateLimitMax: 20, IPHashSalt: "test-salt"}}
	router := newFixtureRouterWithConfig(t, cfg)
	response := performRequest(router, http.MethodPost, "/api/v1/posts/1/likes", "")
	var result posts.LikeResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || !result.Success || result.PostID != 1 || result.Likes != 8 {
		t.Fatalf("unexpected like response: %d %#v", response.Code, result)
	}
	response = performRequest(router, http.MethodPost, "/api/v1/posts/1/likes", "")
	var body errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusTooManyRequests || body.Error.Code != "like_cooldown" || response.Header().Get("Retry-After") == "" {
		t.Fatalf("unexpected cooldown response: %d %#v headers=%v", response.Code, body, response.Header())
	}

	rateRouter := newFixtureRouterWithConfig(t, config.Config{Likes: config.Likes{RateLimitMax: 1, IPHashSalt: "test-salt"}})
	if response := performRequest(rateRouter, http.MethodPost, "/api/v1/posts/1/likes", ""); response.Code != http.StatusOK {
		t.Fatalf("first rate-limited request returned %d", response.Code)
	}
	response = performRequest(rateRouter, http.MethodPost, "/api/v1/posts/2/likes", "")
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusTooManyRequests || body.Error.Code != "like_rate_limited" || response.Header().Get("Retry-After") == "" {
		t.Fatalf("unexpected rate-limit response: %d %#v", response.Code, body)
	}

	crossSite := httptest.NewRequest(http.MethodPost, "/api/v1/posts/2/likes", nil)
	crossSite.Header.Set("Origin", "https://attacker.example")
	crossSiteResponse := httptest.NewRecorder()
	router.ServeHTTP(crossSiteResponse, crossSite)
	if err := json.Unmarshal(crossSiteResponse.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if crossSiteResponse.Code != http.StatusForbidden || body.Error.Code != "cross_site_request" {
		t.Fatalf("cross-site like returned %d %#v", crossSiteResponse.Code, body)
	}

	for _, tt := range []struct {
		path   string
		status int
		code   string
	}{
		{"/api/v1/posts/nope/likes", http.StatusBadRequest, "invalid_post_id"},
		{"/api/v1/posts/999/likes", http.StatusNotFound, "post_not_found"},
	} {
		response = performRequest(router, http.MethodPost, tt.path, "")
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != tt.status || body.Error.Code != tt.code {
			t.Fatalf("%s returned %d %#v", tt.path, response.Code, body)
		}
	}
}

func TestPublicCommentsAPI(t *testing.T) {
	router := newFixtureRouter(t)
	response := performRequest(router, http.MethodGet, "/api/v1/posts/1/comments", "")
	var items []comments.Comment
	if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(items) != 2 || items[0].ID != 10 || items[1].ID != 11 || items[1].ParentID == nil || *items[1].ParentID != 10 {
		t.Fatalf("unexpected comments response: %d %#v", response.Code, items)
	}
	if strings.Contains(response.Body.String(), "pending") || strings.Contains(response.Body.String(), "rejected") || strings.Contains(response.Body.String(), "moderation_reason") || strings.Contains(response.Body.String(), "text_hash") {
		t.Fatalf("comments response leaked hidden data: %s", response.Body.String())
	}

	response = performRequest(router, http.MethodGet, "/api/v1/posts/2/comments", "")
	if response.Code != http.StatusOK || response.Body.String() != "[]" {
		t.Fatalf("hidden-only comments response: %d %s", response.Code, response.Body.String())
	}
	for _, tt := range []struct {
		path   string
		status int
		code   string
	}{
		{"/api/v1/posts/nope/comments", http.StatusBadRequest, "invalid_post_id"},
		{"/api/v1/posts/999/comments", http.StatusNotFound, "post_not_found"},
	} {
		response = performRequest(router, http.MethodGet, tt.path, "")
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != tt.status || body.Error.Code != tt.code {
			t.Fatalf("%s returned %d %#v", tt.path, response.Code, body)
		}
	}
}

func TestCommentChallengeAPI(t *testing.T) {
	cfg := config.Config{Comments: config.Comments{ChallengeSalt: "test-challenge-salt", ChallengeTTL: 30 * time.Minute, ChallengeClockSkew: time.Minute, AttemptRateLimitWindow: time.Minute, AttemptRateLimitMax: 20}}
	router := newFixtureRouterWithConfig(t, cfg)
	response := performRequest(router, http.MethodGet, "/api/v1/posts/1/comments/challenge", "")
	var challenge antispam.Challenge
	if err := json.Unmarshal(response.Body.Bytes(), &challenge); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || challenge.Token == "" || challenge.HoneypotField == "" || challenge.ExpiresInSeconds != 1800 {
		t.Fatalf("unexpected challenge response: %d %#v", response.Code, challenge)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("challenge cache policy = %q", response.Header().Get("Cache-Control"))
	}
	if _, err := antispam.New(nil, cfg.Comments).VerifyChallenge(challenge.Token, 1); err != nil {
		t.Fatalf("HTTP challenge cannot be verified: %v", err)
	}
	if strings.Contains(response.Body.String(), "salt") || strings.Contains(response.Body.String(), "token_hash") {
		t.Fatalf("challenge response leaked internals: %s", response.Body.String())
	}
	for _, tt := range []struct {
		path   string
		status int
		code   string
	}{
		{"/api/v1/posts/nope/comments/challenge", http.StatusBadRequest, "invalid_post_id"},
		{"/api/v1/posts/999/comments/challenge", http.StatusNotFound, "post_not_found"},
	} {
		response = performRequest(router, http.MethodGet, tt.path, "")
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != tt.status || body.Error.Code != tt.code {
			t.Fatalf("%s returned %d %#v", tt.path, response.Code, body)
		}
	}

	rateConfig := cfg
	rateConfig.Comments.AttemptRateLimitMax = 1
	rateRouter := newFixtureRouterWithConfig(t, rateConfig)
	if response := performRequest(rateRouter, http.MethodGet, "/api/v1/posts/1/comments/challenge", ""); response.Code != http.StatusOK {
		t.Fatalf("first challenge request returned %d", response.Code)
	}
	response = performRequest(rateRouter, http.MethodGet, "/api/v1/posts/1/comments/challenge", "")
	var body errorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusTooManyRequests || body.Error.Code != "comment_rate_limited" || response.Header().Get("Retry-After") == "" {
		t.Fatalf("challenge rate limit returned %d %#v", response.Code, body)
	}
}

func TestCommentCreateAPI(t *testing.T) {
	commentConfig := func() config.Config {
		return config.Config{
			Likes: config.Likes{IPHashSalt: "test-salt"},
			Comments: config.Comments{
				MaxNameLength:                     80,
				MaxLength:                         1000,
				MaxURLCount:                       4,
				MaxTokenLength:                    120,
				MaxRepeatedCharRun:                18,
				MaxRepeatedSymbolRun:              10,
				MaxRepeatedTokenRun:               12,
				RandomTextMinLength:               120,
				RandomTokenMinLength:              12,
				RandomTokenMinCount:               4,
				RandomTokenMinShare:               0.5,
				LowTokenDiversityMinTokenCount:    24,
				LowTokenDiversityContentMinLength: 180,
				LowTokenDiversityThreshold:        0.14,
				AttemptRateLimitWindow:            time.Minute,
				AttemptRateLimitMax:               40,
				Cooldown:                          12 * time.Second,
				BurstWindow:                       time.Minute,
				BurstMax:                          6,
				DuplicateWindow:                   3 * time.Minute,
				PostRateLimitWindow:               2 * time.Minute,
				PostRateLimitMax:                  30,
				GlobalRateLimitWindow:             time.Minute,
				GlobalRateLimitMax:                120,
				AttemptsTTL:                       24 * time.Hour,
				ChallengeSalt:                     "test-challenge-salt",
				ChallengeTTL:                      30 * time.Minute,
				ChallengeClockSkew:                time.Minute,
				MuteDuration:                      30 * time.Minute,
				HoneypotMuteThreshold:             2,
				RejectedMuteThreshold:             12,
				AttemptContentMaxLength:           500,
				AdminListLimit:                    40,
			},
			HTTP: config.HTTP{JSONBodyLimit: 256 << 10},
		}
	}

	t.Run("anonymous comment visible", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 1)
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "", "hello world", "", nil))
		var created commentCreateResponse
		if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusCreated || !created.OK || created.Status != "visible" {
			t.Fatalf("unexpected create response: %d %#v", response.Code, created)
		}
		response = performRequest(router, http.MethodGet, "/api/v1/posts/1/comments", "")
		var items []comments.Comment
		if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 3 || items[2].Content != "hello world" || items[2].Name != nil {
			t.Fatalf("created comment not readable: %#v", items)
		}
	})

	t.Run("named reply visible", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 1)
		parent := int64(10)
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "Reader Two", "a quiet reply", "", &parent))
		if response.Code != http.StatusCreated {
			t.Fatalf("reply returned %d %s", response.Code, response.Body.String())
		}
		response = performRequest(router, http.MethodGet, "/api/v1/posts/1/comments", "")
		var items []comments.Comment
		if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		reply := items[len(items)-1]
		if reply.ParentID == nil || *reply.ParentID != 10 || reply.Name == nil || *reply.Name != "Reader Two" {
			t.Fatalf("reply not stored correctly: %#v", reply)
		}
	})

	t.Run("cannot reply across posts", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 2)
		parent := int64(10)
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/2/comments",
			commentCreateBody(challenge, "", "cross post reply", "", &parent))
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusNotFound || body.Error.Code != "parent_comment_not_found" {
			t.Fatalf("cross-post reply returned %d %#v", response.Code, body)
		}
	})

	t.Run("suspicious comment becomes pending", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 1)
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "https://spam.example", "Buy now https://a.example https://b.example cash", "", nil))
		var created commentCreateResponse
		if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusAccepted || created.Status != "pending" {
			t.Fatalf("pending comment returned %d %#v", response.Code, created)
		}
		response = performRequest(router, http.MethodGet, "/api/v1/posts/1/comments", "")
		var items []comments.Comment
		if err := json.Unmarshal(response.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 2 {
			t.Fatalf("pending comment leaked into public list: %s", response.Body.String())
		}
	})

	t.Run("invalid challenge rejected", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(antispam.Challenge{Token: "garbage", HoneypotField: "hp_000000000000"}, "nobody", "hello", "", nil))
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusTooManyRequests || body.Error.Code != "comment_rate_limited" {
			t.Fatalf("invalid challenge returned %d %#v", response.Code, body)
		}
	})

	t.Run("challenge replay rejected", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 1)
		first := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "", "once only", "", nil))
		if first.Code != http.StatusCreated {
			t.Fatalf("first request returned %d %s", first.Code, first.Body.String())
		}
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "", "replay attempt", "", nil))
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusTooManyRequests || body.Error.Code != "comment_rate_limited" {
			t.Fatalf("replay returned %d %#v", response.Code, body)
		}
	})

	t.Run("honeypot rejected", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 1)
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "", "hello world", "spam-site.example", nil))
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusTooManyRequests || body.Error.Code != "comment_rate_limited" {
			t.Fatalf("honeypot returned %d %#v", response.Code, body)
		}
	})

	t.Run("honeypot repeat mutes until retry", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 1)
		for i := 0; i < 2; i++ {
			response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
				commentCreateBody(challenge, "", "spam payload", "spam-site.example", nil))
			if response.Code != http.StatusTooManyRequests {
				t.Fatalf("honeypot hit %d returned %d", i, response.Code)
			}
			challenge = fetchCommentChallenge(t, router, 1)
		}
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "", "a real comment", "", nil))
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusTooManyRequests || body.Error.Code != "comment_rate_limited" || response.Header().Get("Retry-After") == "" {
			t.Fatalf("muted submit returned %d %#v headers=%v", response.Code, body, response.Header())
		}
	})

	t.Run("name too long", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 1)
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, strings.Repeat("n", 81), "hello", "", nil))
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusBadRequest || body.Error.Code != "invalid_name" {
			t.Fatalf("long name returned %d %#v", response.Code, body)
		}
	})

	t.Run("empty content rejected", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 1)
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "", "", "", nil))
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusBadRequest || body.Error.Code != "invalid_content" {
			t.Fatalf("empty content returned %d %#v", response.Code, body)
		}
	})

	t.Run("malformed json rejected", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments", "{")
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusBadRequest || body.Error.Code != "invalid_body" {
			t.Fatalf("malformed body returned %d %#v", response.Code, body)
		}
	})

	t.Run("oversized json rejected", func(t *testing.T) {
		cfg := commentConfig()
		cfg.HTTP.JSONBodyLimit = 1024
		router := newFixtureRouterWithConfig(t, cfg)
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			`{"content":"`+strings.Repeat("a", 4096)+`"}`)
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusBadRequest || body.Error.Code != "invalid_body" {
			t.Fatalf("oversized body returned %d %#v", response.Code, body)
		}
	})

	t.Run("non-json content type rejected", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		response := performRequest(router, http.MethodPost, "/api/v1/posts/1/comments", `{"content":"hello"}`)
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusUnsupportedMediaType || body.Error.Code != "invalid_content_type" {
			t.Fatalf("non-json returned %d %#v", response.Code, body)
		}
	})

	t.Run("invalid post id rejected", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/nope/comments", `{}`)
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusBadRequest || body.Error.Code != "invalid_post_id" {
			t.Fatalf("invalid post returned %d %#v", response.Code, body)
		}
	})

	t.Run("cooldown rate limit after success", func(t *testing.T) {
		router := newFixtureRouterWithConfig(t, commentConfig())
		challenge := fetchCommentChallenge(t, router, 1)
		first := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "", "first message", "", nil))
		if first.Code != http.StatusCreated {
			t.Fatalf("first request returned %d %s", first.Code, first.Body.String())
		}
		challenge = fetchCommentChallenge(t, router, 1)
		response := performJSONRequest(router, http.MethodPost, "/api/v1/posts/1/comments",
			commentCreateBody(challenge, "", "second message", "", nil))
		var body errorResponse
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != http.StatusTooManyRequests || body.Error.Code != "comment_rate_limited" || response.Header().Get("Retry-After") == "" {
			t.Fatalf("cooldown returned %d %#v headers=%v", response.Code, body, response.Header())
		}
	})
}

func fetchCommentChallenge(t *testing.T, router http.Handler, postID int64) antispam.Challenge {
	t.Helper()
	response := performRequest(router, http.MethodGet,
		"/api/v1/posts/"+strconv.FormatInt(postID, 10)+"/comments/challenge", "")
	if response.Code != http.StatusOK {
		t.Fatalf("challenge fetch returned %d %s", response.Code, response.Body.String())
	}
	var challenge antispam.Challenge
	if err := json.Unmarshal(response.Body.Bytes(), &challenge); err != nil {
		t.Fatal(err)
	}
	if challenge.Token == "" || challenge.HoneypotField == "" {
		t.Fatalf("empty challenge: %#v", challenge)
	}
	return challenge
}

func commentCreateBody(challenge antispam.Challenge, name, content, website string, parentID *int64) string {
	payload := map[string]any{
		"name": name, "content": content, "challenge_token": challenge.Token,
		"website": website, challenge.HoneypotField: "",
	}
	if parentID != nil {
		payload["parent_id"] = *parentID
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}

func performJSONRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
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
		{"canonical IPv6", config.Config{Server: config.Server{TrustProxy: true, TrustedProxyCIDR: []string{"10.0.0.0/8"}}}, "2001:db8::1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newTestRouter(t, tt.cfg)
			router.GET("/test/ip", func(c *gin.Context) { c.String(http.StatusOK, ClientIP(c)) })
			request := httptest.NewRequest(http.MethodGet, "/test/ip", nil)
			request.RemoteAddr = "10.0.0.2:1234"
			forwarded := "203.0.113.9"
			if tt.name == "canonical IPv6" {
				forwarded = "2001:0db8:0:0:0:0:0:1"
			}
			request.Header.Set("X-Forwarded-For", forwarded)
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

func newFixtureRouter(t *testing.T) *gin.Engine {
	return newFixtureRouterWithConfig(t, config.Config{})
}

func newFixtureRouterWithConfig(t *testing.T, cfg config.Config) *gin.Engine {
	t.Helper()
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
	router, err := NewRouter(cfg, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)), db)
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
