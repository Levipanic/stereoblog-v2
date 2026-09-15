package posts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

const likeEventRetention = 14 * 24 * time.Hour

type CooldownError struct {
	RetryAfter time.Duration
}

func (e *CooldownError) Error() string {
	return "post was liked too recently"
}

type LikeResult struct {
	Success bool  `json:"success"`
	PostID  int64 `json:"post_id"`
	Likes   int   `json:"likes"`
}

func HashIP(salt, ip string) string {
	sum := sha256.Sum256([]byte(salt + ":" + ip))
	return hex.EncodeToString(sum[:])
}

func (r *Repository) Like(ctx context.Context, postID int64, ipHash string, cooldown time.Duration, now time.Time) (result LikeResult, err error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return result, fmt.Errorf("acquire like connection: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return result, fmt.Errorf("begin post like: %w", err)
	}
	defer func() {
		if err != nil {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	var likes int
	if err = conn.QueryRowContext(ctx, "SELECT likes_count FROM posts WHERE id = ?", postID).Scan(&likes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, ErrNotFound
		}
		return result, fmt.Errorf("find post for like: %w", err)
	}
	var recent string
	err = conn.QueryRowContext(ctx, `SELECT created_at FROM like_events
		WHERE post_id = ? AND ip_hash = ? ORDER BY datetime(created_at) DESC, id DESC LIMIT 1`, postID, ipHash).Scan(&recent)
	if err == nil {
		recentAt, parseErr := database.ParseTime(recent)
		if parseErr == nil && now.Before(recentAt.Add(cooldown)) {
			return result, &CooldownError{RetryAfter: recentAt.Add(cooldown).Sub(now)}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return result, fmt.Errorf("check post like cooldown: %w", err)
	}
	err = nil
	createdAt := database.FormatTime(now)
	if _, err = conn.ExecContext(ctx, "INSERT INTO like_events (post_id, ip_hash, created_at) VALUES (?, ?, ?)", postID, ipHash, createdAt); err != nil {
		return result, fmt.Errorf("record post like: %w", err)
	}
	if err = conn.QueryRowContext(ctx, "UPDATE posts SET likes_count = likes_count + 1 WHERE id = ? RETURNING likes_count", postID).Scan(&likes); err != nil {
		return result, fmt.Errorf("increment post likes: %w", err)
	}
	retention := likeEventRetention
	if cooldown > retention {
		retention = cooldown
	}
	cutoff := database.FormatTime(now.Add(-retention))
	if _, err = conn.ExecContext(ctx, "DELETE FROM like_events WHERE datetime(created_at) < datetime(?)", cutoff); err != nil {
		return result, fmt.Errorf("clean old post likes: %w", err)
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return result, fmt.Errorf("commit post like: %w", err)
	}
	return LikeResult{Success: true, PostID: postID, Likes: max(0, likes)}, nil
}
