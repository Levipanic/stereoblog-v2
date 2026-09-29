package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/backup"
)

func (a *adminAPI) backupHandler(db *sql.DB) gin.HandlerFunc {
	var active sync.Mutex
	limiter := newFixedWindowLimiter(time.Minute, 1)
	return func(c *gin.Context) {
		if !active.TryLock() {
			setRetryAfter(c, time.Minute)
			writeError(c, 429, "backup_busy", "A backup is already running.")
			return
		}
		defer active.Unlock()
		if ok, retry := limiter.allow("backup"); !ok {
			setRetryAfter(c, retry)
			writeError(c, 429, "backup_rate_limited", "Please wait before creating another backup.")
			return
		}
		// Backup generation/download can exceed the ordinary server's 30-second deadline.
		if err := http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(15 * time.Minute)); err != nil && !errors.Is(err, http.ErrNotSupported) {
			a.failed(c, err)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Minute)
		defer cancel()
		archive, err := backup.Create(ctx, db, a.cfg.Storage.UploadsPath)
		if a.failed(c, err) {
			return
		}
		defer archive.Close()
		file, err := os.Open(archive.Path)
		if a.failed(c, err) {
			return
		}
		defer file.Close()
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", `attachment; filename="stereodamage-backup.zip"`)
		a.logger.Info("backup ready", "posts", archive.Manifest.Posts, "comments", archive.Manifest.Comments, "media_files", archive.Manifest.MediaFiles)
		http.ServeContent(c.Writer, c.Request, "stereodamage-backup.zip", time.Time{}, file)
	}
}
