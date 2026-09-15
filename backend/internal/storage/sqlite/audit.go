package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Levipanic/stereoblog-v2/backend/internal/content"
)

type AuditReport struct {
	MigrationVersion     int
	ExpectedVersion      int
	Posts                int
	Comments             int
	PostLikes            int
	CommentLikes         int
	PostLikeEvents       int
	CommentLikeEvents    int
	ForeignKeyViolations int
	CrossPostParents     int
	MediaReferences      int
	UnknownBlocks        int
	UnparseablePosts     []int64
	InvalidPreviewPosts  []int64
	MissingUploads       []string
}

func (r AuditReport) Compatible() bool {
	return r.MigrationVersion == r.ExpectedVersion &&
		r.ForeignKeyViolations == 0 && r.CrossPostParents == 0 &&
		len(r.UnparseablePosts) == 0 && len(r.InvalidPreviewPosts) == 0 && len(r.MissingUploads) == 0
}

func Audit(ctx context.Context, databasePath, uploadsPath string) (AuditReport, error) {
	report := AuditReport{ExpectedVersion: len(migrations)}
	absolutePath, err := filepath.Abs(databasePath)
	if err != nil {
		return report, fmt.Errorf("resolve database path: %w", err)
	}
	info, err := os.Stat(absolutePath)
	if err != nil {
		return report, fmt.Errorf("open database read-only: %w", err)
	}
	if !info.Mode().IsRegular() {
		return report, fmt.Errorf("open database read-only: %s is not a regular file", absolutePath)
	}

	dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}
	query := dsn.Query()
	query.Set("mode", "ro")
	query.Set("_foreign_keys", "on")
	query.Set("_busy_timeout", fmt.Sprint(busyTimeoutMS))
	dsn.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return report, fmt.Errorf("open database read-only: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return report, databaseError("open database read-only", err)
	}
	if _, err := Inspect(ctx, db); err != nil {
		return report, err
	}
	applied, err := appliedMigrations(ctx, db, migrations)
	if err != nil {
		return report, err
	}
	report.MigrationVersion = len(applied)
	if len(applied) != len(migrations) {
		return report, fmt.Errorf("database migration version is %d, expected %d; migrate a disposable copy before auditing", len(applied), len(migrations))
	}
	if err := validateMigratedSchema(ctx, db, applied); err != nil {
		return report, err
	}
	if err := integrityCheck(ctx, db); err != nil {
		return report, err
	}

	if err := db.QueryRowContext(ctx, `SELECT
		(SELECT count(*) FROM posts),
		(SELECT count(*) FROM comments),
		(SELECT coalesce(sum(likes_count), 0) FROM posts),
		(SELECT coalesce(sum(likes_count), 0) FROM comments),
		(SELECT count(*) FROM like_events),
		(SELECT count(*) FROM comment_like_events)
	`).Scan(&report.Posts, &report.Comments, &report.PostLikes, &report.CommentLikes, &report.PostLikeEvents, &report.CommentLikeEvents); err != nil {
		return report, fmt.Errorf("count database records: %w", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM pragma_foreign_key_check").Scan(&report.ForeignKeyViolations); err != nil {
		return report, fmt.Errorf("check foreign keys: %w", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM comments child
		JOIN comments parent ON parent.id = child.parent_id
		WHERE child.post_id <> parent.post_id`).Scan(&report.CrossPostParents); err != nil {
		return report, fmt.Errorf("check comment parent relationships: %w", err)
	}

	rows, err := db.QueryContext(ctx, "SELECT id, blocks_json, preview_media FROM posts ORDER BY id")
	if err != nil {
		return report, fmt.Errorf("read posts: %w", err)
	}
	defer rows.Close()
	references := make([]string, 0)
	for rows.Next() {
		var id int64
		var raw string
		var preview sql.NullString
		if err := rows.Scan(&id, &raw, &preview); err != nil {
			return report, fmt.Errorf("read post: %w", err)
		}
		blocks, err := content.ParseBlocksJSON(raw)
		if err != nil {
			report.UnparseablePosts = append(report.UnparseablePosts, id)
		} else {
			for _, block := range blocks {
				switch block.Type {
				case content.Media:
					references = append(references, block.Src)
				case content.Unknown:
					report.UnknownBlocks++
				}
			}
		}
		if preview.Valid {
			src, err := parsePreviewMedia(preview.String)
			if err != nil {
				report.InvalidPreviewPosts = append(report.InvalidPreviewPosts, id)
			} else {
				references = append(references, src)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return report, fmt.Errorf("read posts: %w", err)
	}
	report.MediaReferences = len(references)
	report.MissingUploads = missingUploads(uploadsPath, references)
	return report, nil
}

func parsePreviewMedia(raw string) (string, error) {
	preview, err := content.ParsePreviewMediaJSON(raw)
	if err != nil {
		return "", err
	}
	return preview.Src, nil
}

func missingUploads(uploadsPath string, references []string) []string {
	missing := make(map[string]struct{})
	for _, reference := range references {
		decoded, err := url.PathUnescape(strings.TrimPrefix(reference, "/uploads/"))
		if err != nil {
			missing[reference] = struct{}{}
			continue
		}
		info, err := os.Stat(filepath.Join(uploadsPath, filepath.FromSlash(decoded)))
		if err != nil || !info.Mode().IsRegular() {
			missing[reference] = struct{}{}
		}
	}
	result := make([]string, 0, len(missing))
	for reference := range missing {
		result = append(result, reference)
	}
	slices.Sort(result)
	return result
}
