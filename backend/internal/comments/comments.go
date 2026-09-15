package comments

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

var ErrPostNotFound = errors.New("post not found")

type Repository struct {
	db *sql.DB
}

type Comment struct {
	ID        int64   `json:"id"`
	PostID    int64   `json:"post_id"`
	ParentID  *int64  `json:"parent_id"`
	Name      *string `json:"name"`
	Content   string  `json:"content"`
	Likes     int     `json:"likes"`
	CreatedAt string  `json:"created_at"`
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ByPostID(ctx context.Context, postID int64) ([]Comment, error) {
	var exists int
	if err := r.db.QueryRowContext(ctx, "SELECT 1 FROM posts WHERE id = ?", postID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("find post for comments: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, post_id, parent_id, name, content, likes_count, created_at
		FROM comments WHERE post_id = ? AND status = 'visible'
		ORDER BY datetime(created_at) ASC, id ASC`, postID)
	if err != nil {
		return nil, fmt.Errorf("query comments: %w", err)
	}
	defer rows.Close()
	comments := []Comment{}
	for rows.Next() {
		var comment Comment
		var parentID sql.NullInt64
		var name sql.NullString
		if err := rows.Scan(&comment.ID, &comment.PostID, &parentID, &name, &comment.Content, &comment.Likes, &comment.CreatedAt); err != nil {
			return nil, fmt.Errorf("read comment: %w", err)
		}
		createdAt, err := database.ParseTime(comment.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("comment %d has invalid created_at", comment.ID)
		}
		comment.CreatedAt = createdAt.Format(time.RFC3339)
		if comment.Likes < 0 {
			comment.Likes = 0
		}
		if parentID.Valid {
			comment.ParentID = &parentID.Int64
		}
		if name.Valid {
			comment.Name = &name.String
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("query comments: %w", err)
	}
	return comments, nil
}
