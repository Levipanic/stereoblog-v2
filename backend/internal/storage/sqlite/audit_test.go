package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
)

func TestAuditMigratedFixtureWithoutModification(t *testing.T) {
	path := testfixture.V1Database(t)
	db, schema, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(context.Background(), db, path, schema, false); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	uploads := fixtureUploads(t)
	before := fileHash(t, path)

	report, err := Audit(context.Background(), path, uploads)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Compatible() {
		t.Fatalf("fixture audit failed: %#v", report)
	}
	if report.Posts != 2 || report.Comments != 4 || report.PostLikes != 7 || report.CommentLikes != 2 {
		t.Fatalf("unexpected content counts: %#v", report)
	}
	if report.PostLikeEvents != 1 || report.CommentLikeEvents != 1 || report.MediaReferences != 6 || report.UnknownBlocks != 2 {
		t.Fatalf("unexpected compatibility counts: %#v", report)
	}
	if after := fileHash(t, path); after != before {
		t.Fatal("read-only audit changed the database file")
	}
}

func TestAuditReportsCriticalContentFailures(t *testing.T) {
	path := testfixture.V1Database(t)
	db, schema, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(context.Background(), db, path, schema, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE posts SET blocks_json = 'not json' WHERE id = 2;
		UPDATE posts SET preview_media = '{}' WHERE id = 1;
		UPDATE comments SET post_id = 2 WHERE id = 11`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	uploads := fixtureUploads(t)
	if err := os.Remove(filepath.Join(uploads, "fixture-video.mp4")); err != nil {
		t.Fatal(err)
	}

	report, err := Audit(context.Background(), path, uploads)
	if err != nil {
		t.Fatal(err)
	}
	if report.Compatible() {
		t.Fatal("critical compatibility failures passed")
	}
	if !reflect.DeepEqual(report.UnparseablePosts, []int64{2}) || !reflect.DeepEqual(report.InvalidPreviewPosts, []int64{1}) {
		t.Fatalf("content failures = %#v", report)
	}
	if report.CrossPostParents != 1 || !reflect.DeepEqual(report.MissingUploads, []string{"/uploads/fixture-video.mp4"}) {
		t.Fatalf("relationship/media failures = %#v", report)
	}
}

func fixtureUploads(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	for _, name := range []string{"fixture-image.jpg", "fixture-animation.gif", "fixture-video.mp4", "fixture-audio.mp3", "fixture-file.zip"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}
