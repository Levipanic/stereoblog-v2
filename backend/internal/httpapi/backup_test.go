package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBackupHTTPAndRestoredRuntime(t *testing.T) {
	r, _, cfg := adminTestRouter(t)
	temporary := t.TempDir()
	t.Setenv("TMPDIR", temporary)
	if res := adminRequest(r, "POST", "/api/v1/admin/backup", "", nil, ""); res.Code != 401 {
		t.Fatal("unauthorized backup")
	}
	cookie, csrf := adminLogin(t, r)
	if res := adminRequest(r, "POST", "/api/v1/admin/backup", "", cookie, ""); res.Code != 403 {
		t.Fatal("missing CSRF accepted")
	}
	res := adminRequest(r, "POST", "/api/v1/admin/posts", `{"title":"Restore me","blocks":[{"type":"paragraph","text":"Saved content"}]}`, cookie, csrf)
	if res.Code != 201 {
		t.Fatal(res.Body.String())
	}
	res = adminRequest(r, "POST", "/api/v1/admin/backup", "", cookie, csrf)
	if res.Code != 200 || res.Header().Get("Content-Type") != "application/zip" || res.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("backup: %d %s", res.Code, res.Body.String())
	}
	z, err := zip.NewReader(bytes.NewReader(res.Body.Bytes()), int64(res.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	restore := t.TempDir()
	for _, entry := range z.File {
		path := filepath.Join(restore, entry.Name)
		os.MkdirAll(filepath.Dir(path), 0700)
		source, err := entry.Open()
		if err != nil {
			t.Fatal(err)
		}
		dest, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(dest, source)
		source.Close()
		dest.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	if res := adminRequest(r, "POST", "/api/v1/admin/backup", "", cookie, csrf); res.Code != 429 {
		t.Fatal("backup rate limit absent")
	}
	entries, err := os.ReadDir(temporary)
	if err != nil || len(entries) != 0 {
		t.Fatal("HTTP backup leaked temporary files")
	}
	dbPath := filepath.Join(restore, "data", "blog.db")
	db, schema, err := database.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := database.Migrate(context.Background(), db, dbPath, schema, false); err != nil {
		t.Fatal(err)
	}
	cfg.Storage.DatabasePath = dbPath
	cfg.Storage.UploadsPath = filepath.Join(restore, "uploads")
	// Exercise the same router constructor as production startup against restored data.
	restored, err := NewRouter(cfg, slog.Default(), db)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(restored)
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/api/v1/posts/restore-me")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var post struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(response.Body).Decode(&post); err != nil || response.StatusCode != 200 || post.Title != "Restore me" {
		t.Fatalf("restored runtime: %#v %v", post, err)
	}
}

type blockedBackupWriter struct {
	*httptest.ResponseRecorder
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (w *blockedBackupWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.started) })
	<-w.release
	return 0, io.ErrClosedPipe // Simulate a disconnected download.
}

func TestBackupSerializesDownloadsAndCleansDisconnect(t *testing.T) {
	r, _, _ := adminTestRouter(t)
	cookie, csrf := adminLogin(t, r)
	temporary := t.TempDir()
	t.Setenv("TMPDIR", temporary)
	w := &blockedBackupWriter{ResponseRecorder: httptest.NewRecorder(), started: make(chan struct{}), release: make(chan struct{})}
	request := httptest.NewRequest("POST", "/api/v1/admin/backup", nil)
	request.AddCookie(cookie)
	request.Header.Set("X-CSRF-Token", csrf)
	done := make(chan struct{})
	go func() { defer close(done); r.ServeHTTP(w, request) }()
	defer func() {
		close(w.release)
		<-done
		entries, err := os.ReadDir(temporary)
		if err != nil || len(entries) != 0 {
			t.Errorf("disconnected download leaked temporary files: %v %v", entries, err)
		}
	}()
	select {
	case <-w.started:
	case <-time.After(5 * time.Second):
		t.Fatal("backup did not start")
	}
	res := adminRequest(r, "POST", "/api/v1/admin/backup", "", cookie, csrf)
	if res.Code != http.StatusTooManyRequests || !strings.Contains(res.Body.String(), "backup_busy") {
		t.Fatalf("parallel backup: %d %s", res.Code, res.Body.String())
	}
	if res := performRequest(r, "GET", "/api/v1/health", ""); res.Code != 200 {
		t.Fatal("backup blocked health")
	}
}
