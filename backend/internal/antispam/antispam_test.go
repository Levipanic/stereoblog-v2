package antispam

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

func TestChallengeIssueVerifyExpiryAndReplay(t *testing.T) {
	db := testDatabase(t)
	cfg := testConfig()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	service := New(db, cfg)
	service.now = func() time.Time { return now }
	service.rand = bytes.NewReader(make([]byte, 18))
	challenge, err := service.IssueChallenge(1)
	if err != nil {
		t.Fatal(err)
	}
	if challenge.HoneypotField != "hp_000000000000" || challenge.ExpiresInSeconds != 1800 || len(strings.Split(challenge.Token, ".")) != 5 {
		t.Fatalf("unexpected challenge: %#v", challenge)
	}
	verified, err := service.VerifyChallenge(challenge.Token, 1)
	if err != nil || verified.PostID != 1 || verified.HoneypotField != challenge.HoneypotField || len(verified.TokenHash) != 64 {
		t.Fatalf("verification = %#v, %v", verified, err)
	}
	if _, err := service.VerifyChallenge(challenge.Token, 2); challengeReason(err) != "invalid_challenge" {
		t.Fatalf("post binding error = %v", err)
	}
	replacement := "0"
	if strings.HasSuffix(challenge.Token, "0") {
		replacement = "1"
	}
	tampered := challenge.Token[:len(challenge.Token)-1] + replacement
	if _, err := service.VerifyChallenge(tampered, 1); challengeReason(err) != "bad_challenge_signature" {
		t.Fatalf("tampered challenge error = %v", err)
	}
	service.now = func() time.Time { return now.Add(cfg.ChallengeTTL + time.Second) }
	if _, err := service.VerifyChallenge(challenge.Token, 1); challengeReason(err) != "expired_challenge" {
		t.Fatalf("expired challenge error = %v", err)
	}
	service.now = func() time.Time { return now }

	start := sync.WaitGroup{}
	start.Add(1)
	errorsOut := make(chan error, 2)
	for range 2 {
		go func() {
			start.Wait()
			errorsOut <- service.ConsumeChallenge(context.Background(), verified)
		}()
	}
	start.Done()
	var success, replay int
	for range 2 {
		err := <-errorsOut
		if err == nil {
			success++
		} else if challengeReason(err) == "challenge_replay" {
			replay++
		} else {
			t.Fatalf("consume error = %v", err)
		}
	}
	if success != 1 || replay != 1 {
		t.Fatalf("challenge consumption: success=%d replay=%d", success, replay)
	}
}

func TestContentValidationHashesAndModeration(t *testing.T) {
	service := New(nil, testConfig())
	human := "Thanks for this post - the audio section was genuinely useful."
	if content, err := service.ValidateContent(human); err != nil || content != human {
		t.Fatalf("human comment rejected: %q, %v", content, err)
	}
	for name, value := range map[string]string{
		"empty":            "   ",
		"too many links":   "https://a.test https://b.test https://c.test https://d.test https://e.test",
		"repeated text":    strings.Repeat("a", 19),
		"repeated symbols": strings.Repeat("🙂", 11),
		"random tokens":    strings.Repeat("aB3_xYzQwErTy9 ", 9),
	} {
		if _, err := service.ValidateContent(value); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
	if _, err := service.ValidateContent("\ufeff"); err == nil {
		t.Fatal("ECMAScript BOM whitespace was accepted as content")
	}
	if content, err := service.ValidateContent("\u0085"); err != nil || content != "\u0085" {
		t.Fatalf("non-ECMAScript whitespace changed: %q, %v", content, err)
	}
	if _, err := service.ValidateContent(strings.Repeat("〰", 11)); err == nil {
		t.Fatal("extended pictographic repetition was accepted")
	}
	if got := truncateUTF16(strings.Repeat("a", 499)+"🙂", 500); !strings.HasSuffix(got, "�") || UTF16Length(got) != 500 {
		t.Fatalf("UTF-16 truncation = %q", got[len(got)-4:])
	}
	if TextHash(" Hello\nWORLD ") != TextHash("hello world") {
		t.Fatal("normalized text hashes differ")
	}
	if Fingerprint("Buy https://a.test/x 123!!!") != Fingerprint("buy www.b.test/y 999??") {
		t.Fatal("similar comments have different fingerprints")
	}

	visible := service.Moderate(RequestSource{Host: "blog.test", Origin: "https://blog.test"}, "", human, Stats{})
	if visible.Status != Visible || visible.Score != 0 || visible.Reason != "" {
		t.Fatalf("human moderation = %#v", visible)
	}
	pending := service.Moderate(RequestSource{Host: "blog.test"}, "", "https://a.test https://b.test", Stats{})
	if pending.Status != Pending || pending.Score != 5 || pending.Reason != "score:5; missing_origin_referer,multiple_links,short_link_comment" {
		t.Fatalf("pending moderation = %#v", pending)
	}
	cross := SourceSignals(RequestSource{Host: "blog.test", Origin: "https://evil.test", Referer: "https://evil.test/x"})
	if len(cross) != 2 || cross[0].Score+cross[1].Score != 5 {
		t.Fatalf("cross-origin signals = %#v", cross)
	}
}

func TestAttemptStatsRateLimitsAndMutes(t *testing.T) {
	db := testDatabase(t)
	cfg := testConfig()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	service := New(db, cfg)
	service.now = func() time.Time { return now }
	postID := int64(1)
	if err := service.RecordAttempt(context.Background(), Attempt{IPHash: "ip", PostID: &postID, Status: Visible, Content: "first", TextHash: "other", Fingerprint: "same"}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(cfg.Cooldown + time.Second)
	stats, err := service.AttemptStats(context.Background(), "ip", 1, "current", "same")
	if err != nil {
		t.Fatal(err)
	}
	if stats.IPRecentCount != 1 || stats.PostRecentCount != 1 || stats.GlobalRecentCount != 1 || stats.FingerprintRecentCount != 1 {
		t.Fatalf("attempt stats = %#v", stats)
	}
	if rate := service.RateLimit(Stats{LastAttemptAt: timePointer(now.Add(-time.Second)), IPRecentCount: cfg.BurstMax}); !rate.Limited || rate.Reason != "cooldown" {
		t.Fatalf("cooldown priority = %#v", rate)
	}
	if rate := service.RateLimit(Stats{IPRecentCount: cfg.BurstMax}); !rate.Limited || rate.Reason != "ip_burst" || rate.RetryAfter != cfg.BurstWindow {
		t.Fatalf("burst decision = %#v", rate)
	}
	if rate := service.RateLimit(Stats{DuplicateAt: timePointer(now.Add(-time.Minute))}); !rate.Limited || rate.Reason != "duplicate" {
		t.Fatalf("duplicate decision = %#v", rate)
	}

	if mute, err := service.RecordHoneypot(context.Background(), Attempt{IPHash: "bot", PostID: &postID}); err != nil || mute != nil {
		t.Fatalf("first honeypot = %#v, %v", mute, err)
	}
	mute, err := service.RecordHoneypot(context.Background(), Attempt{IPHash: "bot", PostID: &postID})
	if err != nil || mute == nil || mute.Reason != "honeypot" || mute.MuteCount != 1 {
		t.Fatalf("second honeypot = %#v, %v", mute, err)
	}
	now = now.Add(cfg.MuteDuration + time.Second)
	if mute, err := service.ActiveMute(context.Background(), "bot"); err != nil || mute != nil {
		t.Fatalf("expired mute = %#v, %v", mute, err)
	}
}

func testConfig() config.Comments {
	return config.Comments{
		MaxLength: 1000, MaxURLCount: 4, MaxTokenLength: 120,
		MaxRepeatedCharRun: 18, MaxRepeatedSymbolRun: 10, MaxRepeatedTokenRun: 12,
		RandomTextMinLength: 120, RandomTokenMinLength: 12, RandomTokenMinCount: 4, RandomTokenMinShare: 0.5,
		LowTokenDiversityMinTokenCount: 24, LowTokenDiversityContentMinLength: 180, LowTokenDiversityThreshold: 0.14,
		Cooldown: 12 * time.Second, BurstWindow: time.Minute, BurstMax: 6,
		DuplicateWindow: 3 * time.Minute, PostRateLimitWindow: 2 * time.Minute, PostRateLimitMax: 30,
		GlobalRateLimitWindow: time.Minute, GlobalRateLimitMax: 120, AttemptsTTL: 24 * time.Hour,
		ChallengeSalt: "challenge-salt", ChallengeTTL: 30 * time.Minute, ChallengeClockSkew: time.Minute,
		MuteDuration: 30 * time.Minute, HoneypotMuteThreshold: 2, RejectedMuteThreshold: 12, AttemptContentMaxLength: 500,
	}
}

func testDatabase(t *testing.T) *sql.DB {
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

func challengeReason(err error) string {
	var challenge *ChallengeError
	if errors.As(err, &challenge) {
		return challenge.Reason
	}
	return ""
}

func timePointer(value time.Time) *time.Time { return &value }
