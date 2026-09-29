package posts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Levipanic/stereoblog-v2/backend/internal/antispam"
	"github.com/Levipanic/stereoblog-v2/backend/internal/content"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

var ErrSlugConflict = errors.New("slug already exists")
var ErrSlugImmutable = errors.New("published slug cannot be changed")

type WriteInput struct {
	Title   string          `json:"title"`
	Slug    string          `json:"slug"`
	Blocks  json.RawMessage `json:"blocks"`
	Preview json.RawMessage `json:"preview_media"`
}
type AdminPost struct {
	ID        int64           `json:"id"`
	Title     string          `json:"title"`
	Slug      string          `json:"slug"`
	Blocks    json.RawMessage `json:"blocks"`
	Preview   json.RawMessage `json:"preview_media"`
	CreatedAt string          `json:"created_at"`
}

func (in *WriteInput) Validate(limits content.Limits) error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || antispam.UTF16Length(in.Title) > 160 {
		return errors.New("title must contain 1–160 characters")
	}
	if in.Slug != "" && (len([]rune(in.Slug)) > 200 || database.Slugify(in.Slug) != in.Slug) {
		return errors.New("slug must be a lowercase, hyphen-separated slug of at most 200 characters")
	}
	blocks, err := content.ValidateBlocksJSON(string(in.Blocks), limits)
	if err != nil {
		return err
	}
	canonical, err := content.MarshalBlocksJSON(blocks)
	if err != nil {
		return err
	}
	in.Blocks = json.RawMessage(canonical)
	if len(in.Preview) > 0 && string(in.Preview) != "null" {
		preview, err := content.ParsePreviewMediaJSON(string(in.Preview))
		if err != nil {
			return errors.New("preview_media must be valid local media")
		}
		for _, value := range []string{preview.Name, preview.Alt, preview.Caption} {
			if antispam.UTF16Length(value) > limits.MaxMediaText {
				return errors.New("preview_media text is too long")
			}
		}
		in.Preview, err = json.Marshal(preview)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) AdminByID(ctx context.Context, id int64) (AdminPost, error) {
	var p AdminPost
	var raw string
	var preview sql.NullString
	err := r.db.QueryRowContext(ctx, "SELECT id, title, slug, blocks_json, preview_media, created_at FROM posts WHERE id = ?", id).Scan(&p.ID, &p.Title, &p.Slug, &raw, &preview, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	if !json.Valid([]byte(raw)) || preview.Valid && !json.Valid([]byte(preview.String)) {
		return p, errors.New("stored post JSON is invalid; refusing lossy admin read")
	}
	p.Blocks = json.RawMessage(raw)
	p.Preview = json.RawMessage("null")
	if preview.Valid {
		p.Preview = json.RawMessage(preview.String)
	}
	return p, nil
}

func (r *Repository) Create(ctx context.Context, in WriteInput) (int64, error) {
	base := in.Slug
	if base == "" {
		base = database.Slugify(in.Title)
	}
	slug := base
	for suffix := 2; ; suffix++ {
		var id int64
		err := r.db.QueryRowContext(ctx, `INSERT INTO posts (title, slug, blocks_json, preview_media, created_at) VALUES (?, ?, ?, ?, datetime('now')) ON CONFLICT(slug) DO NOTHING RETURNING id`, in.Title, slug, string(in.Blocks), previewValue(in.Preview)).Scan(&id)
		if !errors.Is(err, sql.ErrNoRows) {
			return id, err
		}
		if in.Slug != "" {
			return 0, ErrSlugConflict
		}
		slug = fmt.Sprintf("%s-%d", base, suffix)
	}
}

func (r *Repository) Update(ctx context.Context, id int64, in WriteInput) error {
	// Slugs are immutable after publication; omission of preview preserves the stored selection.
	result, err := r.db.ExecContext(ctx, `UPDATE posts SET title = ?, blocks_json = ?, preview_media = CASE WHEN ? THEN preview_media ELSE ? END WHERE id = ? AND (? = '' OR slug = ?)`, in.Title, string(in.Blocks), len(in.Preview) == 0, previewValue(in.Preview), id, in.Slug, in.Slug)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		if _, err := r.SlugByID(ctx, id); err != nil {
			return err
		}
		return ErrSlugImmutable
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM posts WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err == nil && n == 0 {
		return ErrNotFound
	}
	return err
}
func previewValue(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return string(raw)
}
