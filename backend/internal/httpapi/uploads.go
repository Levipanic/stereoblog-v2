package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/Levipanic/stereoblog-v2/backend/internal/media"
	"github.com/gin-gonic/gin"
)

func (a *adminAPI) upload(c *gin.Context) {
	limit := a.cfg.Upload.MaxSize
	if limit <= 0 {
		writeError(c, 500, "upload_unavailable", "Uploads are not configured.")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit+min(limit, 64<<10))
	reader, err := c.Request.MultipartReader()
	if err != nil {
		writeError(c, 415, "invalid_content_type", "Expected multipart/form-data.")
		return
	}
	part, err := reader.NextPart()
	if err != nil {
		writeError(c, 400, "invalid_upload", "One file is required.")
		return
	}
	defer part.Close()
	_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	name := params["filename"]
	if err != nil || part.FormName() != "file" || !media.ValidName(name) {
		writeError(c, 400, "invalid_upload", "One file with a safe filename is required.")
		return
	}
	if err := os.MkdirAll(a.cfg.Storage.UploadsPath, 0750); a.failed(c, err) {
		return
	}
	root, err := os.OpenRoot(a.cfg.Storage.UploadsPath)
	if a.failed(c, err) {
		return
	}
	defer root.Close()
	var random [24]byte
	if _, err := rand.Read(random[:]); a.failed(c, err) {
		return
	}
	token := hex.EncodeToString(random[:])
	temporary := ".upload-" + token
	stored := token + media.SafeExtension(name)
	f, err := root.OpenFile(temporary, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0640)
	if a.failed(c, err) {
		return
	}
	defer f.Close()
	defer root.Remove(temporary)
	n, err := io.Copy(f, io.LimitReader(part, limit+1))
	var tooLarge *http.MaxBytesError
	if n > limit || errors.As(err, &tooLarge) {
		writeError(c, 413, "upload_too_large", "File exceeds upload limit.")
		return
	}
	if err != nil || n == 0 {
		writeError(c, 400, "invalid_upload", "File is empty or incomplete.")
		return
	}
	if extra, err := reader.NextPart(); err != io.EOF {
		if extra != nil {
			extra.Close()
		}
		writeError(c, 400, "invalid_upload", "Exactly one file is allowed.")
		return
	}
	kind, width, height, err := media.Inspect(f, name)
	if err != nil {
		writeError(c, 415, "unsupported_file", "Unsupported file or file type does not match its extension. New SVG uploads are disabled.")
		return
	}
	if a.failed(c, f.Sync()) || a.failed(c, f.Close()) || a.failed(c, root.Rename(temporary, stored)) {
		return
	}
	c.JSON(201, media.Upload{URL: "/uploads/" + stored, OriginalName: name, StoredName: stored, MediaKind: kind, Width: width, Height: height})
}

func serveUpload(directory string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := strings.TrimPrefix(c.Param("file"), "/")
		root, err := os.OpenRoot(directory)
		if err != nil {
			writeError(c, 404, "not_found", "File not found.")
			return
		}
		defer root.Close()
		file, err := media.OpenRegular(root, name)
		if err != nil {
			writeError(c, 404, "not_found", "File not found.")
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			writeError(c, 404, "not_found", "File not found.")
			return
		}
		c.Header("Content-Security-Policy", "sandbox; default-src 'none'; style-src 'unsafe-inline'")
		typ := media.InlineType(name)
		if typ == "" {
			typ = "application/octet-stream"
			c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": path.Base(name)}))
		}
		c.Header("Content-Type", typ)
		http.ServeContent(c.Writer, c.Request, path.Base(name), info.ModTime(), file)
	}
}
