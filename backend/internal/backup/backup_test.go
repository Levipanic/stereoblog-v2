package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
)

func backupFixture(t *testing.T) (*sql.DB, string) {
	t.Helper()
	ctx := context.Background()
	path := testfixture.V1Database(t)
	db, schema, err := database.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := database.Migrate(ctx, db, path, schema, false); err != nil {
		t.Fatal(err)
	}
	uploads := t.TempDir()
	for _, name := range []string{"fixture-image.jpg", "fixture-animation.gif", "fixture-video.mp4", "fixture-audio.mp3", "fixture-file.zip"} {
		if err := os.WriteFile(filepath.Join(uploads, name), []byte("original bytes "+name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return db, uploads
}

func TestBackupRestorePreservesV1DataAndMedia(t *testing.T) {
	db, uploads := backupFixture(t)
	ctx := context.Background()
	os.WriteFile(filepath.Join(uploads, ".env"), []byte("PRIVATE_SECRET"), 0600)
	os.WriteFile(filepath.Join(uploads, ".upload-in-progress"), []byte("partial upload"), 0600)
	archive, err := Create(ctx, db, uploads)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	info, err := os.Stat(archive.Path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("archive permissions")
	}
	z, err := zip.OpenReader(archive.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	restore := t.TempDir()
	for _, file := range z.File {
		if strings.Contains(file.Name, ".env") || strings.Contains(file.Name, ".upload-") || strings.Contains(file.Name, "..") {
			t.Fatalf("unsafe entry %s", file.Name)
		}
		dest := filepath.Join(restore, filepath.FromSlash(file.Name))
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			t.Fatal(err)
		}
		src, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		out, err := os.Create(dest)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(out, src)
		src.Close()
		out.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	var manifest Manifest
	raw, err := os.ReadFile(filepath.Join(restore, "manifest.json"))
	if err != nil || json.Unmarshal(raw, &manifest) != nil {
		t.Fatal("invalid manifest")
	}
	if manifest.FormatVersion != 1 || manifest.SchemaVersion != 2 || manifest.Posts != 2 || manifest.Comments != 4 || manifest.MediaFiles != 5 {
		t.Fatalf("manifest %#v", manifest)
	}
	snapshot := filepath.Join(restore, "data", "blog.db")
	raw, err = os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != manifest.DatabaseSHA256 {
		t.Fatal("snapshot checksum mismatch")
	}
	report, err := database.Audit(ctx, snapshot, filepath.Join(restore, "uploads"))
	if err != nil || !report.Compatible() {
		t.Fatalf("restore audit: %#v %v", report, err)
	}
	restored, schema, err := database.Open(ctx, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if migration, err := database.Migrate(ctx, restored, snapshot, schema, false); err != nil || migration.Applied != 0 {
		t.Fatalf("restored startup migration: %#v %v", migration, err)
	}
	for _, table := range []string{"posts", "comments", "like_events", "comment_like_events", "comment_attempts", "comment_mutes", "comment_challenge_uses", "admin_sessions", "schema_migrations"} {
		if !reflect.DeepEqual(tableRows(t, db, table), tableRows(t, restored, table)) {
			t.Fatalf("restored table changed: %s", table)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(restore, "uploads"))
	for _, entry := range entries {
		original, _ := os.ReadFile(filepath.Join(uploads, entry.Name()))
		copy, _ := os.ReadFile(filepath.Join(restore, "uploads", entry.Name()))
		if string(original) != string(copy) {
			t.Fatal("media changed")
		}
	}
	archive.Close()
	if _, err := os.Stat(archive.directory); !os.IsNotExist(err) {
		t.Fatal("temporary resources leaked")
	}
}

func tableRows(t *testing.T, db *sql.DB, table string) [][]any {
	t.Helper()
	rows, err := db.Query("SELECT * FROM " + table + " ORDER BY 1")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, _ := rows.Columns()
	result := [][]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		targets := make([]any, len(columns))
		for i := range values {
			targets[i] = &values[i]
		}
		if err := rows.Scan(targets...); err != nil {
			t.Fatal(err)
		}
		result = append(result, values)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestBackupFailuresCleanUp(t *testing.T) {
	db, uploads := backupFixture(t)
	temporary := t.TempDir()
	t.Setenv("TMPDIR", temporary)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Create(ctx, db, uploads); err == nil {
		t.Fatal("cancelled backup succeeded")
	}
	outside := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(outside, []byte("secret"), 0600)
	link := filepath.Join(uploads, "leak.txt")
	os.Symlink(outside, link)
	if _, err := Create(context.Background(), db, uploads); err == nil {
		t.Fatal("symlink accepted")
	}
	os.Remove(link)
	os.Remove(filepath.Join(uploads, "fixture-image.jpg"))
	if _, err := Create(context.Background(), db, uploads); err == nil {
		t.Fatal("missing media silently accepted")
	}
	entries, err := os.ReadDir(temporary)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed backup leaked files: %v %v", entries, err)
	}
}

func TestBackupWhileWritingIsConsistent(t *testing.T) {
	db, uploads := backupFixture(t)
	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		for range 20 {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				done <- err
				return
			}
			_, err = tx.Exec("UPDATE posts SET likes_count=likes_count+1 WHERE id=1; INSERT INTO like_events(post_id,ip_hash) VALUES (1,'concurrent')")
			if err != nil {
				tx.Rollback()
				done <- err
				return
			}
			if err := tx.Commit(); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	archive, err := Create(ctx, db, uploads)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	report, err := database.Audit(ctx, filepath.Join(archive.directory, "blog.db"), uploads)
	if err != nil || !report.Compatible() || report.PostLikes-report.PostLikeEvents != 6 {
		t.Fatalf("inconsistent snapshot: %#v %v", report, err)
	}
}
