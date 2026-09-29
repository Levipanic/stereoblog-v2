package comments

import (
	"context"
	"errors"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
	"testing"
	"time"
)

func TestCommentLikes(t *testing.T) {
	db := migratedFixture(t)
	r := NewRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	hash := posts.HashIP("salt", "192.0.2.1")
	for _, id := range []int64{12, 13, 999} {
		if _, err := r.Like(ctx, id, hash, time.Minute, now); !errors.Is(err, ErrNotFound) {
			t.Fatalf("hidden/missing %d: %v", id, err)
		}
	}
	results := make(chan error, 2)
	for range 2 {
		go func() { _, err := r.Like(ctx, 10, hash, time.Minute, now); results <- err }()
	}
	success, limited := 0, 0
	for range 2 {
		err := <-results
		var cooldown *posts.CooldownError
		if err == nil {
			success++
		} else if errors.As(err, &cooldown) {
			limited++
		} else {
			t.Fatal(err)
		}
	}
	var count, events int
	if err := db.QueryRow("SELECT likes_count FROM comments WHERE id = 10").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM comment_like_events WHERE ip_hash = ?", hash).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if success != 1 || limited != 1 || count != 3 || events != 1 {
		t.Fatalf("success=%d limited=%d count=%d events=%d", success, limited, count, events)
	}
	if _, err := r.Like(ctx, 10, hash, time.Minute, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TRIGGER fail_like BEFORE UPDATE ON comments BEGIN SELECT RAISE(ABORT, 'test'); END"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Like(ctx, 10, "rollback", time.Minute, now); err == nil {
		t.Fatal("expected failure")
	}
	if err := db.QueryRow("SELECT count(*) FROM comment_like_events WHERE ip_hash = 'rollback'").Scan(&events); err != nil || events != 0 {
		t.Fatalf("event was not rolled back: %d %v", events, err)
	}
}
