package posts

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Levipanic/stereoblog-v2/backend/internal/content"
)

const (
	DefaultLimit        = 10
	MaxLimit            = 50
	commentPreviewLimit = 2
	storageTimeFormat   = "2006-01-02 15:04:05"
)

var ErrInvalidCursor = errors.New("invalid cursor")

type Repository struct {
	db *sql.DB
}

type FeedPage struct {
	Items      []FeedItem `json:"items"`
	NextCursor *string    `json:"next_cursor"`
}

type FeedItem struct {
	ID              int64                 `json:"id"`
	Slug            string                `json:"slug"`
	Title           string                `json:"title"`
	CreatedAt       string                `json:"created_at"`
	Likes           int                   `json:"likes"`
	ReadingMinutes  int                   `json:"reading_minutes"`
	PreviewText     string                `json:"preview_text"`
	PreviewMedia    *content.PreviewMedia `json:"preview_media"`
	CommentCount    int                   `json:"comment_count"`
	CommentPreviews []CommentPreview      `json:"comment_previews"`
	cursorCreatedAt string
}

type CommentPreview struct {
	ID        int64   `json:"id"`
	ParentID  *int64  `json:"parent_id"`
	Name      *string `json:"name"`
	Content   string  `json:"content"`
	CreatedAt string  `json:"created_at"`
}

type cursor struct {
	CreatedAt string `json:"created_at"`
	ID        int64  `json:"id"`
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Feed(ctx context.Context, encodedCursor string, limit int) (FeedPage, error) {
	page := FeedPage{Items: []FeedItem{}}
	boundary, err := decodeCursor(encodedCursor)
	if err != nil {
		return page, err
	}
	query := `SELECT id, slug, title, blocks_json, preview_media, likes_count, created_at FROM posts`
	args := []any{}
	if boundary != nil {
		query += ` WHERE datetime(created_at) < datetime(?) OR (datetime(created_at) = datetime(?) AND id < ?)`
		args = append(args, boundary.CreatedAt, boundary.CreatedAt, boundary.ID)
	}
	query += ` ORDER BY datetime(created_at) DESC, id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return page, fmt.Errorf("query feed posts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item FeedItem
		var raw string
		var preview sql.NullString
		if err := rows.Scan(&item.ID, &item.Slug, &item.Title, &raw, &preview, &item.Likes, &item.CreatedAt); err != nil {
			return page, fmt.Errorf("read feed post: %w", err)
		}
		createdAt, err := parseStorageTime(item.CreatedAt)
		if err != nil {
			return page, fmt.Errorf("post %d has invalid created_at", item.ID)
		}
		item.cursorCreatedAt = item.CreatedAt
		item.CreatedAt = createdAt.Format(time.RFC3339)
		if item.Likes < 0 {
			item.Likes = 0
		}
		blocks, _ := content.ParseBlocksJSON(raw)
		item.PreviewText, item.ReadingMinutes, item.PreviewMedia = content.FeedSummary(blocks)
		if preview.Valid {
			if explicit, err := content.ParsePreviewMediaJSON(preview.String); err == nil {
				item.PreviewMedia = explicit
			}
		}
		item.CommentPreviews = []CommentPreview{}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return page, fmt.Errorf("query feed posts: %w", err)
	}
	if err := rows.Close(); err != nil {
		return page, fmt.Errorf("query feed posts: %w", err)
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		value, err := encodeCursor(page.Items[len(page.Items)-1])
		if err != nil {
			return page, err
		}
		page.NextCursor = &value
	}
	if err := r.addCommentPreviews(ctx, page.Items); err != nil {
		return page, err
	}
	return page, nil
}

func (r *Repository) addCommentPreviews(ctx context.Context, items []FeedItem) error {
	if len(items) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(items)), ",")
	args := make([]any, len(items))
	byID := make(map[int64]int, len(items))
	for index := range items {
		args[index] = items[index].ID
		byID[items[index].ID] = index
	}
	query := fmt.Sprintf(`WITH visible AS (
		SELECT id, post_id, parent_id, name, content, created_at,
			count(*) OVER (PARTITION BY post_id) AS comment_count,
			row_number() OVER (PARTITION BY post_id ORDER BY datetime(created_at) DESC, id DESC) AS preview_rank
		FROM comments WHERE status = 'visible' AND post_id IN (%s)
	) SELECT id, post_id, parent_id, name, content, created_at, comment_count
	FROM visible WHERE preview_rank <= %d ORDER BY post_id, preview_rank`, placeholders, commentPreviewLimit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("query feed comment previews: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var preview CommentPreview
		var postID int64
		var parentID sql.NullInt64
		var name sql.NullString
		var count int
		if err := rows.Scan(&preview.ID, &postID, &parentID, &name, &preview.Content, &preview.CreatedAt, &count); err != nil {
			return fmt.Errorf("read feed comment preview: %w", err)
		}
		if parentID.Valid {
			preview.ParentID = &parentID.Int64
		}
		if name.Valid {
			preview.Name = &name.String
		}
		createdAt, err := parseStorageTime(preview.CreatedAt)
		if err != nil {
			return fmt.Errorf("comment %d has invalid created_at", preview.ID)
		}
		preview.CreatedAt = createdAt.Format(time.RFC3339)
		index := byID[postID]
		items[index].CommentCount = count
		items[index].CommentPreviews = append(items[index].CommentPreviews, preview)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("query feed comment previews: %w", err)
	}
	return nil
}

func decodeCursor(value string) (*cursor, error) {
	if value == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var result cursor
	if json.Unmarshal(raw, &result) != nil || result.ID <= 0 {
		return nil, ErrInvalidCursor
	}
	if _, err := parseStorageTime(result.CreatedAt); err != nil {
		return nil, ErrInvalidCursor
	}
	return &result, nil
}

func parseStorageTime(value string) (time.Time, error) {
	for _, layout := range []string{storageTimeFormat, "2006-01-02 15:04:05.999999999"} {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil {
			return parsed, nil
		}
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, errors.New("unsupported SQLite timestamp")
}

func encodeCursor(item FeedItem) (string, error) {
	raw, err := json.Marshal(cursor{CreatedAt: item.cursorCreatedAt, ID: item.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
