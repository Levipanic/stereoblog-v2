package antispam

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
)

var (
	noncePattern     = regexp.MustCompile(`(?i)^[a-f0-9]{24}$`)
	honeypotPattern  = regexp.MustCompile(`(?i)^hp_[a-f0-9]{12}$`)
	signaturePattern = regexp.MustCompile(`(?i)^[a-f0-9]{64}$`)
)

type Service struct {
	cfg                config.Comments
	db                 *sql.DB
	now                func() time.Time
	rand               io.Reader
	cleanupMu          sync.Mutex
	lastAttemptCleanup time.Time
}

type Challenge struct {
	Token            string `json:"token"`
	HoneypotField    string `json:"honeypot_field"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

type VerifiedChallenge struct {
	PostID        int64
	IssuedAt      time.Time
	HoneypotField string
	TokenHash     string
	Age           time.Duration
}

type ChallengeError struct {
	Reason string
}

func (e *ChallengeError) Error() string { return e.Reason }

func New(db *sql.DB, cfg config.Comments) *Service {
	return &Service{cfg: cfg, db: db, now: time.Now, rand: rand.Reader}
}

func (s *Service) IssueChallenge(postID int64) (Challenge, error) {
	if postID <= 0 {
		return Challenge{}, errors.New("post ID must be positive")
	}
	if s.cfg.ChallengeSalt == "" || s.cfg.ChallengeTTL <= 0 || s.cfg.ChallengeClockSkew <= 0 {
		return Challenge{}, errors.New("challenge configuration is invalid")
	}
	nonce := make([]byte, 12)
	honeypot := make([]byte, 6)
	if _, err := io.ReadFull(s.rand, nonce); err != nil {
		return Challenge{}, fmt.Errorf("generate challenge nonce: %w", err)
	}
	if _, err := io.ReadFull(s.rand, honeypot); err != nil {
		return Challenge{}, fmt.Errorf("generate challenge honeypot: %w", err)
	}
	issuedAt := s.now().Unix()
	honeypotField := "hp_" + hex.EncodeToString(honeypot)
	payload := fmt.Sprintf("%d.%d.%s.%s", postID, issuedAt, hex.EncodeToString(nonce), honeypotField)
	return Challenge{
		Token:            payload + "." + s.sign(payload),
		HoneypotField:    honeypotField,
		ExpiresInSeconds: int64(s.cfg.ChallengeTTL / time.Second),
	}, nil
}

func (s *Service) VerifyChallenge(token string, expectedPostID int64) (VerifiedChallenge, error) {
	if s.cfg.ChallengeSalt == "" || s.cfg.ChallengeTTL <= 0 || s.cfg.ChallengeClockSkew <= 0 {
		return VerifiedChallenge{}, errors.New("challenge configuration is invalid")
	}
	value := strings.TrimSpace(token)
	parts := strings.Split(value, ".")
	if len(parts) != 5 {
		return VerifiedChallenge{}, challengeError("invalid_challenge_format")
	}
	postID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || postID <= 0 {
		return VerifiedChallenge{}, challengeError("invalid_challenge_post")
	}
	issuedAtUnix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || issuedAtUnix <= 0 {
		return VerifiedChallenge{}, challengeError("invalid_challenge_time")
	}
	if !noncePattern.MatchString(parts[2]) {
		return VerifiedChallenge{}, challengeError("invalid_challenge_nonce")
	}
	if !honeypotPattern.MatchString(parts[3]) {
		return VerifiedChallenge{}, challengeError("invalid_challenge_honeypot")
	}
	if !signaturePattern.MatchString(parts[4]) {
		return VerifiedChallenge{}, challengeError("invalid_challenge_signature")
	}
	payload := strings.Join(parts[:4], ".")
	expected, _ := hex.DecodeString(s.sign(payload))
	provided, _ := hex.DecodeString(parts[4])
	if !hmac.Equal(provided, expected) {
		return VerifiedChallenge{}, challengeError("bad_challenge_signature")
	}
	if expectedPostID > 0 && postID != expectedPostID {
		return VerifiedChallenge{}, challengeError("invalid_challenge")
	}
	now := s.now()
	issuedAt := time.Unix(issuedAtUnix, 0)
	if issuedAt.After(now.Add(s.cfg.ChallengeClockSkew)) {
		return VerifiedChallenge{}, challengeError("future_challenge")
	}
	if now.Sub(issuedAt) > s.cfg.ChallengeTTL {
		return VerifiedChallenge{}, challengeError("expired_challenge")
	}
	age := now.Sub(issuedAt).Truncate(time.Second)
	if age < 0 {
		age = 0
	}
	return VerifiedChallenge{PostID: postID, IssuedAt: issuedAt, HoneypotField: parts[3], TokenHash: hash(s.cfg.ChallengeSalt + ":" + value), Age: age}, nil
}

func (s *Service) sign(payload string) string {
	signature := hmac.New(sha256.New, []byte(s.cfg.ChallengeSalt))
	_, _ = signature.Write([]byte(payload))
	return hex.EncodeToString(signature.Sum(nil))
}

func challengeError(reason string) error { return &ChallengeError{Reason: reason} }

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
