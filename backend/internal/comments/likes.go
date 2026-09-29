package comments

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

var ErrNotFound = errors.New("comment not found")

type LikeResult struct {
	Success   bool  `json:"success"`
	CommentID int64 `json:"comment_id"`
	Likes     int   `json:"likes"`
}

func (r *Repository) Like(ctx context.Context, id int64, hash string, cooldown time.Duration, now time.Time) (result LikeResult, err error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return result, err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return result, err
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), "ROLLBACK") }()
	var count int
	if err = conn.QueryRowContext(ctx, "SELECT likes_count FROM comments WHERE id = ? AND status = 'visible'", id).Scan(&count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, ErrNotFound
		}
		return result, err
	}
	var recent string
	err = conn.QueryRowContext(ctx, `SELECT created_at FROM comment_like_events WHERE comment_id = ? AND ip_hash = ? ORDER BY datetime(created_at) DESC, id DESC LIMIT 1`, id, hash).Scan(&recent)
	if err == nil {
		if at, e := database.ParseTime(recent); e == nil && now.Before(at.Add(cooldown)) {
			return result, &posts.CooldownError{RetryAfter: at.Add(cooldown).Sub(now)}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	if _, err = conn.ExecContext(ctx, "INSERT INTO comment_like_events (comment_id, ip_hash, created_at) VALUES (?, ?, ?)", id, hash, database.FormatTime(now)); err != nil {
		return result, err
	}
	if err = conn.QueryRowContext(ctx, "UPDATE comments SET likes_count = likes_count + 1 WHERE id = ? RETURNING likes_count", id).Scan(&count); err != nil {
		return result, err
	}
	if _, err = conn.ExecContext(ctx, "DELETE FROM comment_like_events WHERE datetime(created_at) < datetime(?)", database.FormatTime(now.Add(-max(14*24*time.Hour, cooldown)))); err != nil {
		return result, err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return result, err
	}
	return LikeResult{true, id, max(0, count)}, nil
}
