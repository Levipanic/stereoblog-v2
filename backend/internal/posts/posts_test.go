package posts

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/Levipanic/stereoblog-v2/backend/internal/content"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
)

func TestFeedCursorPaginationAndCommentPreviews(t *testing.T) {
	db := migratedFixture(t)
	if _, err := db.Exec(`INSERT INTO posts (id, title, blocks_json, slug, created_at) VALUES
		(3, 'Newest tie', '[{"type":"paragraph","text":"Third post"}]', 'newest-tie', '2024-01-02 03:04:05'),
		(4, 'Older', '[{"type":"paragraph","text":"Older post"}]', 'older', '2024-01-01T03:04:05Z');
		INSERT INTO comments (id, post_id, name, content, created_at, status)
		VALUES (14, 1, '', 'Newest visible', '2024-01-02 04:02:00', 'visible')`); err != nil {
		t.Fatal(err)
	}
	repository := NewRepository(db)

	var ids []int64
	cursor := ""
	for {
		page, err := repository.Feed(context.Background(), cursor, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) != 1 {
			t.Fatalf("page items = %d, want 1", len(page.Items))
		}
		ids = append(ids, page.Items[0].ID)
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if want := []int64{3, 2, 1, 4}; !equalIDs(ids, want) {
		t.Fatalf("paginated IDs = %v, want %v", ids, want)
	}

	page, err := repository.Feed(context.Background(), "", 10)
	if err != nil {
		t.Fatal(err)
	}
	post := page.Items[2]
	if post.ID != 1 || post.Slug != "тестовая-публикация" || post.Likes != 7 || post.ReadingMinutes != 1 {
		t.Fatalf("unexpected post summary: %#v", post)
	}
	if post.PreviewText != "Привет из синтетической фикстуры." || post.PreviewMedia == nil || post.PreviewMedia.MediaKind != "image" {
		t.Fatalf("unexpected content preview: %#v", post)
	}
	if post.CreatedAt != "2024-01-02T03:04:05Z" {
		t.Fatalf("created_at = %q", post.CreatedAt)
	}
	if post.CommentCount != 3 || len(post.CommentPreviews) != 2 || post.CommentPreviews[0].ID != 14 || post.CommentPreviews[1].ID != 11 {
		t.Fatalf("unexpected comment preview: %#v", post)
	}
	if post.CommentPreviews[0].Name == nil || *post.CommentPreviews[0].Name != "" || post.CommentPreviews[1].Name == nil || *post.CommentPreviews[1].Name != "Reader" {
		t.Fatalf("nullable comment names changed: %#v", post.CommentPreviews)
	}
	if _, err := repository.Feed(context.Background(), "not-a-cursor", 10); err != ErrInvalidCursor {
		t.Fatalf("invalid cursor error = %v", err)
	}
}

func TestPostBySlugAndIDResolution(t *testing.T) {
	db := migratedFixture(t)
	repository := NewRepository(db)

	post, err := repository.BySlug(context.Background(), "тестовая-публикация")
	if err != nil {
		t.Fatal(err)
	}
	if post.ID != 1 || post.Slug != "тестовая-публикация" || len(post.Blocks) != 9 || post.Blocks[4].Type != content.Media {
		t.Fatalf("unexpected legacy post: %#v", post)
	}
	if post.PreviewText != "Привет из синтетической фикстуры." || post.PreviewMedia == nil || post.PreviewMedia.MediaKind != content.Image {
		t.Fatalf("unexpected post metadata: %#v", post)
	}

	unknown, err := repository.BySlug(context.Background(), "english-fixture-post")
	if err != nil {
		t.Fatal(err)
	}
	if len(unknown.Blocks) != 3 || unknown.Blocks[1].Type != content.Unknown || unknown.Blocks[2].Type != content.Unknown {
		t.Fatalf("unknown legacy blocks were not handled safely: %#v", unknown.Blocks)
	}
	if _, err := db.Exec("UPDATE posts SET blocks_json = 'not json' WHERE id = 2"); err != nil {
		t.Fatal(err)
	}
	broken, err := repository.BySlug(context.Background(), "english-fixture-post")
	if err != nil || len(broken.Blocks) != 0 {
		t.Fatalf("malformed legacy document did not degrade safely: post=%#v error=%v", broken, err)
	}

	resolved, err := repository.SlugByID(context.Background(), 1)
	if err != nil || resolved.ID != 1 || resolved.Slug != "тестовая-публикация" {
		t.Fatalf("ID resolution = %#v, %v", resolved, err)
	}
	if _, err := repository.BySlug(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing slug error = %v", err)
	}
	if _, err := repository.SlugByID(context.Background(), 999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing ID error = %v", err)
	}
}

func migratedFixture(t *testing.T) *sql.DB {
	t.Helper()
	path := testfixture.V1Database(t)
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

func equalIDs(got, want []int64) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
