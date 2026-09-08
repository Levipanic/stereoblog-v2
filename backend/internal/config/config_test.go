package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(mapLookup(nil))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ListenAddress() != "127.0.0.1:8080" {
		t.Fatalf("unexpected listen address: %s", cfg.ListenAddress())
	}
	if cfg.Server.Environment != "development" || cfg.Server.TrustProxy || cfg.IsProduction() {
		t.Fatalf("unexpected server defaults: %#v", cfg.Server)
	}
	if cfg.Storage.DatabasePath != "../data/blog.db" || cfg.Storage.UploadsPath != "../uploads" {
		t.Fatalf("unexpected storage defaults: %#v", cfg.Storage)
	}
	if cfg.HTTP.JSONBodyLimit != 256<<10 || cfg.Upload.MaxSize != 25<<20 {
		t.Fatalf("unexpected size defaults: http=%d upload=%d", cfg.HTTP.JSONBodyLimit, cfg.Upload.MaxSize)
	}
	if cfg.Likes.Cooldown != 10*time.Second || cfg.Comments.AttemptsTTL != 24*time.Hour || cfg.Admin.SessionTTL != 12*time.Hour {
		t.Fatal("duration defaults use the wrong units")
	}
	if cfg.Likes.IPHashSalt != defaultAdminSecret || cfg.Comments.ChallengeSalt != defaultAdminSecret || cfg.Admin.SessionHashSalt != defaultAdminSecret {
		t.Fatal("salt defaults must follow ADMIN_SECRET")
	}
}

func TestLoadOverrides(t *testing.T) {
	env := map[string]string{
		"API_HOST": "::1", "API_PORT": "9090", "APP_ENV": "test",
		"TRUST_PROXY": "true", "TRUSTED_PROXIES": "127.0.0.1/32, ::1/128",
		"DB_PATH": "tmp/blog.db", "UPLOADS_PATH": "tmp/uploads",
		"JSON_BODY_LIMIT": "2kb", "GLOBAL_API_RATE_LIMIT_WINDOW_SECONDS": "2", "GLOBAL_API_RATE_LIMIT_MAX": "2",
		"UPLOAD_MAX_SIZE":       "2mb",
		"LIKE_COOLDOWN_SECONDS": "2", "LIKE_RATE_LIMIT_MAX": "2", "LIKE_IP_HASH_SALT": "like-salt",
		"COMMENT_MAX_NAME_LENGTH": "2", "COMMENT_MAX_LENGTH": "2", "COMMENT_MAX_URL_COUNT": "2",
		"COMMENT_MAX_TOKEN_LENGTH": "2", "COMMENT_MAX_REPEATED_CHAR_RUN": "2", "COMMENT_MAX_REPEATED_SYMBOL_RUN": "2",
		"COMMENT_MAX_REPEATED_TOKEN_RUN": "2", "COMMENT_RANDOM_TEXT_MIN_LENGTH": "2", "COMMENT_RANDOM_TOKEN_MIN_LENGTH": "2",
		"COMMENT_RANDOM_TOKEN_MIN_COUNT": "2", "COMMENT_RANDOM_TOKEN_MIN_SHARE": "0.25",
		"COMMENT_LOW_TOKEN_DIVERSITY_MIN_TOKEN_COUNT": "2", "COMMENT_LOW_TOKEN_DIVERSITY_CONTENT_MIN_LENGTH": "2",
		"COMMENT_LOW_TOKEN_DIVERSITY_THRESHOLD": "0.25", "COMMENT_ATTEMPT_RATE_LIMIT_WINDOW_SECONDS": "2",
		"COMMENT_ATTEMPT_RATE_LIMIT_MAX": "2", "COMMENT_COOLDOWN_SECONDS": "2", "COMMENT_BURST_WINDOW_SECONDS": "2",
		"COMMENT_BURST_MAX": "2", "COMMENT_DUPLICATE_WINDOW_SECONDS": "2", "COMMENT_POST_RATE_LIMIT_WINDOW_SECONDS": "2",
		"COMMENT_POST_RATE_LIMIT_MAX": "2", "COMMENT_GLOBAL_RATE_LIMIT_WINDOW_SECONDS": "2", "COMMENT_GLOBAL_RATE_LIMIT_MAX": "2",
		"COMMENT_ATTEMPTS_TTL_HOURS": "2", "COMMENT_CHALLENGE_SALT": "challenge-salt", "COMMENT_CHALLENGE_TTL_SECONDS": "2",
		"COMMENT_CHALLENGE_CLOCK_SKEW_SECONDS": "2", "COMMENT_MUTE_SECONDS": "2", "COMMENT_HONEYPOT_MUTE_THRESHOLD": "2",
		"COMMENT_REJECTED_MUTE_THRESHOLD": "2", "COMMENT_ATTEMPT_CONTENT_MAX_LENGTH": "2", "COMMENT_ADMIN_LIST_LIMIT": "2",
		"POST_MAX_BLOCKS": "2", "POST_MAX_TEXT_LENGTH": "2", "POST_MAX_MEDIA_TEXT_LENGTH": "2",
		"ADMIN_SECRET": "admin-secret", "ADMIN_POST_RATE_LIMIT_WINDOW_SECONDS": "2", "ADMIN_POST_RATE_LIMIT_MAX": "2",
		"ADMIN_SESSION_TTL_HOURS": "2", "ADMIN_SESSION_HASH_SALT": "session-salt", "ADMIN_SESSION_CLOCK_SKEW_SECONDS": "2",
		"ADMIN_LOGIN_RATE_LIMIT_MAX": "2",
	}

	cfg, err := load(mapLookup(env))
	if err != nil {
		t.Fatal(err)
	}

	want := Config{
		Server:  Server{"::1", 9090, "test", true, []string{"127.0.0.1/32", "::1/128"}},
		Storage: Storage{"tmp/blog.db", "tmp/uploads"},
		HTTP:    HTTP{2 << 10, 2 * time.Second, 2},
		Upload:  Upload{2 << 20},
		Likes:   Likes{2 * time.Second, 2, "like-salt"},
		Comments: Comments{
			MaxNameLength: 2, MaxLength: 2, MaxURLCount: 2, MaxTokenLength: 2,
			MaxRepeatedCharRun: 2, MaxRepeatedSymbolRun: 2, MaxRepeatedTokenRun: 2,
			RandomTextMinLength: 2, RandomTokenMinLength: 2, RandomTokenMinCount: 2, RandomTokenMinShare: 0.25,
			LowTokenDiversityMinTokenCount: 2, LowTokenDiversityContentMinLength: 2, LowTokenDiversityThreshold: 0.25,
			AttemptRateLimitWindow: 2 * time.Second, AttemptRateLimitMax: 2, Cooldown: 2 * time.Second,
			BurstWindow: 2 * time.Second, BurstMax: 2, DuplicateWindow: 2 * time.Second,
			PostRateLimitWindow: 2 * time.Second, PostRateLimitMax: 2,
			GlobalRateLimitWindow: 2 * time.Second, GlobalRateLimitMax: 2, AttemptsTTL: 2 * time.Hour,
			ChallengeSalt: "challenge-salt", ChallengeTTL: 2 * time.Second, ChallengeClockSkew: 2 * time.Second,
			MuteDuration: 2 * time.Second, HoneypotMuteThreshold: 2, RejectedMuteThreshold: 2,
			AttemptContentMaxLength: 2, AdminListLimit: 2,
		},
		Posts: Posts{2, 2, 2},
		Admin: Admin{"admin-secret", 2 * time.Second, 2, 2 * time.Hour, "session-salt", 2 * time.Second, 2},
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("config mismatch\n got: %#v\nwant: %#v", cfg, want)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		key  string
	}{
		{"bad integer", map[string]string{"COMMENT_MAX_LENGTH": "1.5"}, "COMMENT_MAX_LENGTH"},
		{"zero integer", map[string]string{"COMMENT_MAX_LENGTH": "0"}, "COMMENT_MAX_LENGTH"},
		{"bad port", map[string]string{"API_PORT": "65536"}, "API_PORT"},
		{"bad boolean", map[string]string{"TRUST_PROXY": "maybe"}, "TRUST_PROXY"},
		{"missing proxy list", map[string]string{"TRUST_PROXY": "true"}, "TRUSTED_PROXIES"},
		{"bad proxy list", map[string]string{"TRUSTED_PROXIES": "localhost"}, "TRUSTED_PROXIES"},
		{"bad host", map[string]string{"API_HOST": "localhost"}, "API_HOST"},
		{"bad environment", map[string]string{"APP_ENV": "staging"}, "APP_ENV"},
		{"bad bytes", map[string]string{"UPLOAD_MAX_SIZE": "large"}, "UPLOAD_MAX_SIZE"},
		{"bad ratio", map[string]string{"COMMENT_RANDOM_TOKEN_MIN_SHARE": "2"}, "COMMENT_RANDOM_TOKEN_MIN_SHARE"},
		{"nan ratio", map[string]string{"COMMENT_RANDOM_TOKEN_MIN_SHARE": "NaN"}, "COMMENT_RANDOM_TOKEN_MIN_SHARE"},
		{"bad duration", map[string]string{"COMMENT_ATTEMPTS_TTL_HOURS": "-1"}, "COMMENT_ATTEMPTS_TTL_HOURS"},
		{"inconsistent runs", map[string]string{"COMMENT_MAX_REPEATED_CHAR_RUN": "2", "COMMENT_MAX_REPEATED_SYMBOL_RUN": "3"}, "COMMENT_MAX_REPEATED_SYMBOL_RUN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := load(mapLookup(tt.env))
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("expected error naming %s, got %v", tt.key, err)
			}
		})
	}
}

func TestProductionRequiresNonDefaultSecretWithoutLeakingIt(t *testing.T) {
	for _, secret := range []string{"", defaultAdminSecret, " " + defaultAdminSecret + " "} {
		_, err := load(mapLookup(map[string]string{"APP_ENV": "production", "ADMIN_SECRET": secret}))
		if err == nil || !strings.Contains(err.Error(), "ADMIN_SECRET") {
			t.Fatalf("expected ADMIN_SECRET error, got %v", err)
		}
	}

	secret := "do-not-print-this-secret"
	cfg, err := load(mapLookup(map[string]string{"APP_ENV": "production", "ADMIN_SECRET": secret}))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsProduction() || cfg.Admin.Secret != secret {
		t.Fatal("valid production config was not retained")
	}

	_, err = load(mapLookup(map[string]string{"APP_ENV": "production", "ADMIN_SECRET": secret, "ADMIN_SESSION_TTL_HOURS": "bad"}))
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("configuration error leaked secret: %v", err)
	}
}

func TestSaltsInheritConfiguredAdminSecret(t *testing.T) {
	cfg, err := load(mapLookup(map[string]string{"ADMIN_SECRET": "custom-secret"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Likes.IPHashSalt != cfg.Admin.Secret || cfg.Comments.ChallengeSalt != cfg.Admin.Secret || cfg.Admin.SessionHashSalt != cfg.Admin.Secret {
		t.Fatal("empty salts must inherit the configured ADMIN_SECRET")
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
