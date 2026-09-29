package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/Levipanic/stereoblog-v2/backend/internal/media"
	"image"
	"image/gif"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func uploadRequest(t *testing.T, r http.Handler, cookie *http.Cookie, csrf, name string, data []byte, extra bool) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	p, err := w.CreateFormFile("file", name)
	if err != nil {
		t.Fatal(err)
	}
	p.Write(data)
	if extra {
		p, _ = w.CreateFormFile("file", "extra.txt")
		p.Write([]byte("extra"))
	}
	w.Close()
	req := httptest.NewRequest("POST", "/api/v1/admin/uploads", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-CSRF-Token", csrf)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	return res
}
func TestUploadsAndSafeServing(t *testing.T) {
	r, _, cfg := adminTestRouter(t)
	cookie, csrf := adminLogin(t, r)
	var pngData, gifData bytes.Buffer
	png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 2, 3)))
	gif.Encode(&gifData, image.NewRGBA(image.Rect(0, 0, 2, 3)), nil)
	if res := uploadRequest(t, r, nil, "", "image.png", pngData.Bytes(), false); res.Code != 401 {
		t.Fatal(res.Code)
	}
	if res := uploadRequest(t, r, cookie, "", "image.png", pngData.Bytes(), false); res.Code != 403 {
		t.Fatal(res.Code)
	}
	res := uploadRequest(t, r, cookie, csrf, "image.png", pngData.Bytes(), false)
	if res.Code != 201 {
		t.Fatalf("upload: %d %s", res.Code, res.Body.String())
	}
	var result media.Upload
	if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Width != 2 || result.Height != 3 || result.MediaKind != "image" || result.StoredName == "image.png" {
		t.Fatalf("metadata: %#v", result)
	}
	res = performRequest(r, "GET", result.URL, "")
	if res.Code != 200 || !bytes.Equal(res.Body.Bytes(), pngData.Bytes()) {
		t.Fatal("media did not byte-match")
	}
	req := httptest.NewRequest("GET", result.URL, nil)
	req.Header.Set("Range", "bytes=0-3")
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != 206 || res.Body.Len() != 4 {
		t.Fatal("range serving broken")
	}
	res = uploadRequest(t, r, cookie, csrf, "archive.zip", []byte("PK\x03\x04archive"), false)
	if res.Code != 201 {
		t.Fatal(res.Body.String())
	}
	json.Unmarshal(res.Body.Bytes(), &result)
	res = performRequest(r, "GET", result.URL, "")
	if !strings.HasPrefix(res.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal("generic file served inline")
	}
	for _, tc := range []struct {
		name   string
		data   []byte
		extra  bool
		status int
	}{
		{"fake.png", []byte("<html><script>alert(1)</script>"), false, 415},
		{"new.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), false, 415},
		{"empty.txt", nil, false, 400}, {"../escape.txt", []byte("test"), false, 400},
		{"ok.txt", []byte("test"), true, 400}, {"huge.txt", bytes.Repeat([]byte("x"), int(cfg.Upload.MaxSize)+1), false, 413},
	} {
		res := uploadRequest(t, r, cookie, csrf, tc.name, tc.data, tc.extra)
		if res.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.name, res.Code, res.Body.String())
		}
	}
	entries, err := os.ReadDir(cfg.Storage.UploadsPath)
	if err != nil || len(entries) != 2 {
		t.Fatalf("failed upload left files: %d %v", len(entries), err)
	}
	for _, tc := range []struct {
		name string
		data []byte
		kind string
	}{
		{"animation.gif", gifData.Bytes(), "gif"},
		{"track.wav", append([]byte("RIFF\x24\x00\x00\x00WAVEfmt "), make([]byte, 44)...), "audio"},
		{"clip.mp4", []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00mp42isom"), "video"},
	} {
		res := uploadRequest(t, r, cookie, csrf, tc.name, tc.data, false)
		if res.Code != 201 {
			t.Fatalf("%s: %d %s", tc.name, res.Code, res.Body.String())
		}
		var item media.Upload
		json.Unmarshal(res.Body.Bytes(), &item)
		if item.MediaKind != tc.kind {
			t.Fatalf("%s kind %s", tc.name, item.MediaKind)
		}
	}
	if err := os.WriteFile(filepath.Join(cfg.Storage.UploadsPath, "old.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), 0600); err != nil {
		t.Fatal(err)
	}
	res = performRequest(r, "GET", "/uploads/old.svg", "")
	if res.Code != 200 || !strings.Contains(res.Header().Get("Content-Security-Policy"), "sandbox") || res.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("historical SVG not sandboxed")
	}
	outside := filepath.Join(t.TempDir(), "secret.txt")
	os.WriteFile(outside, []byte("secret"), 0600)
	os.Symlink(outside, filepath.Join(cfg.Storage.UploadsPath, "leak.txt"))
	os.WriteFile(filepath.Join(cfg.Storage.UploadsPath, ".env"), []byte("secret"), 0600)
	for _, url := range []string{"/uploads/leak.txt", "/uploads/.env", "/uploads/../secret.txt", "/uploads/"} {
		if res := performRequest(r, "GET", url, ""); res.Code == 200 {
			t.Fatalf("unsafe path served: %s", url)
		}
	}
}
