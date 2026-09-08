package config

import (
	"fmt"
	"math"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const defaultAdminSecret = "change-me"

type Config struct {
	Server   Server
	Storage  Storage
	HTTP     HTTP
	Upload   Upload
	Likes    Likes
	Comments Comments
	Posts    Posts
	Admin    Admin
}

type Server struct {
	Host             string
	Port             int
	Environment      string
	TrustProxy       bool
	TrustedProxyCIDR []string
}

type Storage struct {
	DatabasePath string
	UploadsPath  string
}

type HTTP struct {
	JSONBodyLimit         int64
	GlobalRateLimitWindow time.Duration
	GlobalRateLimitMax    int
}

type Upload struct {
	MaxSize int64
}

type Likes struct {
	Cooldown     time.Duration
	RateLimitMax int
	IPHashSalt   string
}

type Comments struct {
	MaxNameLength                     int
	MaxLength                         int
	MaxURLCount                       int
	MaxTokenLength                    int
	MaxRepeatedCharRun                int
	MaxRepeatedSymbolRun              int
	MaxRepeatedTokenRun               int
	RandomTextMinLength               int
	RandomTokenMinLength              int
	RandomTokenMinCount               int
	RandomTokenMinShare               float64
	LowTokenDiversityMinTokenCount    int
	LowTokenDiversityContentMinLength int
	LowTokenDiversityThreshold        float64
	AttemptRateLimitWindow            time.Duration
	AttemptRateLimitMax               int
	Cooldown                          time.Duration
	BurstWindow                       time.Duration
	BurstMax                          int
	DuplicateWindow                   time.Duration
	PostRateLimitWindow               time.Duration
	PostRateLimitMax                  int
	GlobalRateLimitWindow             time.Duration
	GlobalRateLimitMax                int
	AttemptsTTL                       time.Duration
	ChallengeSalt                     string
	ChallengeTTL                      time.Duration
	ChallengeClockSkew                time.Duration
	MuteDuration                      time.Duration
	HoneypotMuteThreshold             int
	RejectedMuteThreshold             int
	AttemptContentMaxLength           int
	AdminListLimit                    int
}

type Posts struct {
	MaxBlocks          int
	MaxTextLength      int
	MaxMediaTextLength int
}

type Admin struct {
	Secret              string
	PostRateLimitWindow time.Duration
	PostRateLimitMax    int
	SessionTTL          time.Duration
	SessionHashSalt     string
	SessionClockSkew    time.Duration
	LoginRateLimitMax   int
}

func Load() (Config, error) {
	return load(os.LookupEnv)
}

func (c Config) ListenAddress() string {
	return net.JoinHostPort(c.Server.Host, strconv.Itoa(c.Server.Port))
}

func (c Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

type envLoader struct {
	lookup func(string) (string, bool)
	err    error
}

func load(lookup func(string) (string, bool)) (Config, error) {
	l := envLoader{lookup: lookup}

	adminSecret := l.secret("ADMIN_SECRET", defaultAdminSecret)
	cfg := Config{
		Server: Server{
			Host:             l.host("API_HOST", "127.0.0.1"),
			Port:             l.port("API_PORT", 8080),
			Environment:      l.enum("APP_ENV", "development", "development", "test", "production"),
			TrustProxy:       l.boolean("TRUST_PROXY", false),
			TrustedProxyCIDR: l.cidrs("TRUSTED_PROXIES"),
		},
		Storage: Storage{
			DatabasePath: l.path("DB_PATH", "../data/blog.db"),
			UploadsPath:  l.path("UPLOADS_PATH", "../uploads"),
		},
		HTTP: HTTP{
			JSONBodyLimit:         l.bytes("JSON_BODY_LIMIT", "256kb"),
			GlobalRateLimitWindow: l.duration("GLOBAL_API_RATE_LIMIT_WINDOW_SECONDS", 60, time.Second),
			GlobalRateLimitMax:    l.integer("GLOBAL_API_RATE_LIMIT_MAX", 240),
		},
		Upload: Upload{MaxSize: l.bytes("UPLOAD_MAX_SIZE", "25mb")},
		Likes: Likes{
			Cooldown:     l.duration("LIKE_COOLDOWN_SECONDS", 10, time.Second),
			RateLimitMax: l.integer("LIKE_RATE_LIMIT_MAX", 20),
			IPHashSalt:   l.secret("LIKE_IP_HASH_SALT", adminSecret),
		},
		Comments: Comments{
			MaxNameLength:                     l.integer("COMMENT_MAX_NAME_LENGTH", 80),
			MaxLength:                         l.integer("COMMENT_MAX_LENGTH", 1000),
			MaxURLCount:                       l.integer("COMMENT_MAX_URL_COUNT", 4),
			MaxTokenLength:                    l.integer("COMMENT_MAX_TOKEN_LENGTH", 120),
			MaxRepeatedCharRun:                l.integer("COMMENT_MAX_REPEATED_CHAR_RUN", 18),
			MaxRepeatedSymbolRun:              l.integer("COMMENT_MAX_REPEATED_SYMBOL_RUN", 10),
			MaxRepeatedTokenRun:               l.integer("COMMENT_MAX_REPEATED_TOKEN_RUN", 12),
			RandomTextMinLength:               l.integer("COMMENT_RANDOM_TEXT_MIN_LENGTH", 120),
			RandomTokenMinLength:              l.integer("COMMENT_RANDOM_TOKEN_MIN_LENGTH", 12),
			RandomTokenMinCount:               l.integer("COMMENT_RANDOM_TOKEN_MIN_COUNT", 4),
			RandomTokenMinShare:               l.ratio("COMMENT_RANDOM_TOKEN_MIN_SHARE", 0.5),
			LowTokenDiversityMinTokenCount:    l.integer("COMMENT_LOW_TOKEN_DIVERSITY_MIN_TOKEN_COUNT", 24),
			LowTokenDiversityContentMinLength: l.integer("COMMENT_LOW_TOKEN_DIVERSITY_CONTENT_MIN_LENGTH", 180),
			LowTokenDiversityThreshold:        l.ratio("COMMENT_LOW_TOKEN_DIVERSITY_THRESHOLD", 0.14),
			AttemptRateLimitWindow:            l.duration("COMMENT_ATTEMPT_RATE_LIMIT_WINDOW_SECONDS", 60, time.Second),
			AttemptRateLimitMax:               l.integer("COMMENT_ATTEMPT_RATE_LIMIT_MAX", 40),
			Cooldown:                          l.duration("COMMENT_COOLDOWN_SECONDS", 12, time.Second),
			BurstWindow:                       l.duration("COMMENT_BURST_WINDOW_SECONDS", 60, time.Second),
			BurstMax:                          l.integer("COMMENT_BURST_MAX", 6),
			DuplicateWindow:                   l.duration("COMMENT_DUPLICATE_WINDOW_SECONDS", 180, time.Second),
			PostRateLimitWindow:               l.duration("COMMENT_POST_RATE_LIMIT_WINDOW_SECONDS", 120, time.Second),
			PostRateLimitMax:                  l.integer("COMMENT_POST_RATE_LIMIT_MAX", 30),
			GlobalRateLimitWindow:             l.duration("COMMENT_GLOBAL_RATE_LIMIT_WINDOW_SECONDS", 60, time.Second),
			GlobalRateLimitMax:                l.integer("COMMENT_GLOBAL_RATE_LIMIT_MAX", 120),
			AttemptsTTL:                       l.duration("COMMENT_ATTEMPTS_TTL_HOURS", 24, time.Hour),
			ChallengeSalt:                     l.secret("COMMENT_CHALLENGE_SALT", adminSecret),
			ChallengeTTL:                      l.duration("COMMENT_CHALLENGE_TTL_SECONDS", 1800, time.Second),
			ChallengeClockSkew:                l.duration("COMMENT_CHALLENGE_CLOCK_SKEW_SECONDS", 60, time.Second),
			MuteDuration:                      l.duration("COMMENT_MUTE_SECONDS", 1800, time.Second),
			HoneypotMuteThreshold:             l.integer("COMMENT_HONEYPOT_MUTE_THRESHOLD", 2),
			RejectedMuteThreshold:             l.integer("COMMENT_REJECTED_MUTE_THRESHOLD", 12),
			AttemptContentMaxLength:           l.integer("COMMENT_ATTEMPT_CONTENT_MAX_LENGTH", 500),
			AdminListLimit:                    l.integer("COMMENT_ADMIN_LIST_LIMIT", 40),
		},
		Posts: Posts{
			MaxBlocks:          l.integer("POST_MAX_BLOCKS", 60),
			MaxTextLength:      l.integer("POST_MAX_TEXT_LENGTH", 4000),
			MaxMediaTextLength: l.integer("POST_MAX_MEDIA_TEXT_LENGTH", 500),
		},
		Admin: Admin{
			Secret:              adminSecret,
			PostRateLimitWindow: l.duration("ADMIN_POST_RATE_LIMIT_WINDOW_SECONDS", 300, time.Second),
			PostRateLimitMax:    l.integer("ADMIN_POST_RATE_LIMIT_MAX", 12),
			SessionTTL:          l.duration("ADMIN_SESSION_TTL_HOURS", 12, time.Hour),
			SessionHashSalt:     l.secret("ADMIN_SESSION_HASH_SALT", adminSecret),
			SessionClockSkew:    l.duration("ADMIN_SESSION_CLOCK_SKEW_SECONDS", 60, time.Second),
			LoginRateLimitMax:   l.integer("ADMIN_LOGIN_RATE_LIMIT_MAX", 6),
		},
	}

	if l.err != nil {
		return Config{}, l.err
	}
	if cfg.Comments.MaxRepeatedSymbolRun > cfg.Comments.MaxRepeatedCharRun {
		return Config{}, fmt.Errorf("COMMENT_MAX_REPEATED_SYMBOL_RUN must not exceed COMMENT_MAX_REPEATED_CHAR_RUN")
	}
	if cfg.Server.TrustProxy && len(cfg.Server.TrustedProxyCIDR) == 0 {
		return Config{}, fmt.Errorf("TRUSTED_PROXIES must be set when TRUST_PROXY is enabled")
	}
	trimmedSecret := strings.TrimSpace(cfg.Admin.Secret)
	if cfg.IsProduction() && (trimmedSecret == "" || trimmedSecret == defaultAdminSecret) {
		return Config{}, fmt.Errorf("ADMIN_SECRET must be set to a non-default value in production")
	}

	return cfg, nil
}

func (l *envLoader) value(name, fallback string) string {
	if l.err != nil {
		return fallback
	}
	value, ok := l.lookup(name)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func (l *envLoader) secret(name, fallback string) string {
	if l.err != nil {
		return fallback
	}
	value, ok := l.lookup(name)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func (l *envLoader) integer(name string, fallback int) int {
	value := l.value(name, strconv.Itoa(fallback))
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		l.fail(name, "must be a positive integer")
		return fallback
	}
	return n
}

func (l *envLoader) port(name string, fallback int) int {
	n := l.integer(name, fallback)
	if n > 65535 {
		l.fail(name, "must be between 1 and 65535")
		return fallback
	}
	return n
}

func (l *envLoader) duration(name string, fallback int64, unit time.Duration) time.Duration {
	value := l.value(name, strconv.FormatInt(fallback, 10))
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 || n > math.MaxInt64/int64(unit) {
		l.fail(name, "must be a positive integer duration")
		return time.Duration(fallback) * unit
	}
	return time.Duration(n) * unit
}

func (l *envLoader) ratio(name string, fallback float64) float64 {
	value := l.value(name, strconv.FormatFloat(fallback, 'g', -1, 64))
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 || n > 1 {
		l.fail(name, "must be a number greater than 0 and at most 1")
		return fallback
	}
	return n
}

func (l *envLoader) boolean(name string, fallback bool) bool {
	value := l.value(name, strconv.FormatBool(fallback))
	result, err := strconv.ParseBool(value)
	if err != nil {
		l.fail(name, "must be a boolean")
		return fallback
	}
	return result
}

func (l *envLoader) bytes(name, fallback string) int64 {
	value := strings.ToLower(l.value(name, fallback))
	multiplier := int64(1)
	for _, unit := range []struct {
		suffix string
		size   int64
	}{{"gb", 1 << 30}, {"mb", 1 << 20}, {"kb", 1 << 10}, {"b", 1}} {
		if strings.HasSuffix(value, unit.suffix) {
			value = strings.TrimSpace(strings.TrimSuffix(value, unit.suffix))
			multiplier = unit.size
			break
		}
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n <= 0 || n > math.MaxInt64/multiplier {
		l.fail(name, "must be a positive byte size using b, kb, mb, or gb")
		return 0
	}
	return n * multiplier
}

func (l *envLoader) enum(name, fallback string, allowed ...string) string {
	value := l.value(name, fallback)
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	l.fail(name, "has an unsupported value")
	return fallback
}

func (l *envLoader) host(name, fallback string) string {
	value := l.value(name, fallback)
	if _, err := netip.ParseAddr(value); err != nil {
		l.fail(name, "must be an IP address")
		return fallback
	}
	return value
}

func (l *envLoader) path(name, fallback string) string {
	return filepath.Clean(l.value(name, fallback))
}

func (l *envLoader) cidrs(name string) []string {
	value, ok := l.lookup(name)
	if !ok || strings.TrimSpace(value) == "" || l.err != nil {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if _, err := netip.ParsePrefix(part); err != nil {
			l.fail(name, "must contain comma-separated CIDR prefixes")
			return nil
		}
		result = append(result, part)
	}
	return result
}

func (l *envLoader) fail(name, message string) {
	if l.err == nil {
		l.err = fmt.Errorf("%s %s", name, message)
	}
}
