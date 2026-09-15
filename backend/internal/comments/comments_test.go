package comments

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
)

func TestVisibleCommentsPreserveRelationshipsAndStableOrder(t *testing.T) {
	db := migratedFixture(t)
	if _, err := db.Exec(`INSERT INTO comments (id, post_id, parent_id, name, content, likes_count, created_at, status)
		VALUES (14, 1, 10, '', 'Equal-time reply', 3, '2024-01-02 04:01:00', 'visible')`); err != nil {
		t.Fatal(err)
	}
	repository := NewRepository(db)
	items, err := repository.ByPostID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].ID != 10 || items[1].ID != 11 || items[2].ID != 14 {
		t.Fatalf("comments are not stably ordered: %#v", items)
	}
	if items[0].Name != nil || items[0].ParentID != nil || items[0].Likes != 2 {
		t.Fatalf("root comment changed: %#v", items[0])
	}
	if items[1].ParentID == nil || *items[1].ParentID != 10 || items[1].Name == nil || *items[1].Name != "Reader" {
		t.Fatalf("nested comment changed: %#v", items[1])
	}
	if items[2].Name == nil || *items[2].Name != "" || items[2].CreatedAt != "2024-01-02T04:01:00Z" {
		t.Fatalf("empty optional name/timestamp changed: %#v", items[2])
	}
	empty, err := repository.ByPostID(context.Background(), 2)
	if err != nil || len(empty) != 0 || empty == nil {
		t.Fatalf("hidden comments leaked: %#v, %v", empty, err)
	}
	if _, err := repository.ByPostID(context.Background(), 999); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("missing post error = %v", err)
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
