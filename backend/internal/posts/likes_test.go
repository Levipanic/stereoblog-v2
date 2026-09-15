package posts

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPostLikeIsAtomicAndEnforcesCooldown(t *testing.T) {
	db := migratedFixture(t)
	repository := NewRepository(db)
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	rawIP := "203.0.113.9"
	hash := HashIP("test-salt", rawIP)
	if len(hash) != 64 || strings.Contains(hash, rawIP) || hash == HashIP("other-salt", rawIP) {
		t.Fatalf("unsafe IP hash %q", hash)
	}
	if _, err := db.Exec("INSERT INTO like_events (post_id, ip_hash, created_at) VALUES (1, 'old', '2020-01-01 00:00:00')"); err != nil {
		t.Fatal(err)
	}

	result, err := repository.Like(context.Background(), 1, hash, 10*time.Second, now)
	if err != nil || !result.Success || result.PostID != 1 || result.Likes != 8 {
		t.Fatalf("like result = %#v, %v", result, err)
	}
	if _, err := repository.Like(context.Background(), 1, hash, 10*time.Second, now.Add(time.Second)); err == nil {
		t.Fatal("cooldown did not reject a repeated like")
	} else {
		var cooldown *CooldownError
		if !errors.As(err, &cooldown) || cooldown.RetryAfter != 9*time.Second {
			t.Fatalf("cooldown error = %#v, %v", cooldown, err)
		}
	}
	var likes, events, old, raw int
	if err := db.QueryRow("SELECT likes_count FROM posts WHERE id = 1").Scan(&likes); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM like_events WHERE ip_hash = ?", hash).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM like_events WHERE ip_hash = 'old'").Scan(&old); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM like_events WHERE ip_hash = ?", rawIP).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if likes != 8 || events != 1 || old != 0 || raw != 0 {
		t.Fatalf("persisted like state: likes=%d events=%d old=%d raw=%d", likes, events, old, raw)
	}
	if _, err := repository.Like(context.Background(), 999, hash, 0, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing post error = %v", err)
	}
}

func TestLikeRetentionNeverShortensCooldown(t *testing.T) {
	db := migratedFixture(t)
	repository := NewRepository(db)
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	hash := HashIP("test-salt", "203.0.113.10")
	createdAt := now.Add(-20 * 24 * time.Hour).Format(storageTimeFormat)
	if _, err := db.Exec("INSERT INTO like_events (post_id, ip_hash, created_at) VALUES (1, ?, ?)", hash, createdAt); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Like(context.Background(), 1, hash, 30*24*time.Hour, now); err == nil {
		t.Fatal("retention shortened configured cooldown")
	} else {
		var cooldown *CooldownError
		if !errors.As(err, &cooldown) || cooldown.RetryAfter != 10*24*time.Hour {
			t.Fatalf("cooldown error = %#v, %v", cooldown, err)
		}
	}
}

func TestConcurrentPostLikesCannotDesyncCountAndEvents(t *testing.T) {
	db := migratedFixture(t)
	repository := NewRepository(db)
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)
	hash := HashIP("test-salt", "198.51.100.7")
	errorsOut := make(chan error, 2)
	var start sync.WaitGroup
	start.Add(1)
	for range 2 {
		go func() {
			start.Wait()
			_, err := repository.Like(context.Background(), 1, hash, time.Minute, now)
			errorsOut <- err
		}()
	}
	start.Done()
	var succeeded, cooledDown int
	for range 2 {
		err := <-errorsOut
		if err == nil {
			succeeded++
			continue
		}
		var cooldown *CooldownError
		if errors.As(err, &cooldown) {
			cooledDown++
			continue
		}
		t.Fatalf("unexpected concurrent error: %v", err)
	}
	var likes, events int
	if err := db.QueryRow("SELECT likes_count FROM posts WHERE id = 1").Scan(&likes); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM like_events WHERE ip_hash = ?", hash).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if succeeded != 1 || cooledDown != 1 || likes != 8 || events != 1 {
		t.Fatalf("concurrent state: success=%d cooldown=%d likes=%d events=%d", succeeded, cooledDown, likes, events)
	}
}
