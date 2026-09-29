package httpapi

import (
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/Levipanic/stereoblog-v2/backend/internal/admin"
	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
	"github.com/gin-gonic/gin"
)

const adminCookie = "admin_session"

type adminAPI struct {
	sessions *admin.Sessions
	cfg      config.Config
	logger   *slog.Logger
}

func registerAdmin(router *gin.Engine, db *sql.DB, cfg config.Config, logger *slog.Logger) {
	a := &adminAPI{admin.NewSessions(db, cfg.Admin), cfg, logger}
	g := router.Group("/api/v1/admin")
	g.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() })
	limiter := newFixedWindowLimiter(15*time.Minute, cfg.Admin.LoginRateLimitMax)
	g.POST("/login", func(c *gin.Context) {
		if crossSiteRequest(c) {
			writeError(c, 403, "cross_site_request", "Cross-site write request rejected.")
			return
		}
		if ok, retry := limiter.allow(ClientIP(c)); !ok {
			setRetryAfter(c, retry)
			writeError(c, 429, "login_rate_limited", "Too many login requests.")
			return
		}
		var input struct {
			Secret string `json:"secret"`
		}
		if !decodeAdminJSON(c, cfg.HTTP.JSONBodyLimit, &input) {
			return
		}
		if !a.sessions.ValidSecret(input.Secret) {
			writeError(c, 401, "invalid_credentials", "Invalid admin secret.")
			return
		}
		token, session, err := a.sessions.Create(c.Request.Context())
		if a.failed(c, err) {
			return
		}
		a.cookie(c, token, int(cfg.Admin.SessionTTL.Seconds()))
		c.JSON(200, session)
	})
	g.GET("/session", func(c *gin.Context) {
		token, _ := c.Cookie(adminCookie)
		session, err := a.sessions.Get(c.Request.Context(), token)
		if a.failed(c, err) {
			return
		}
		c.JSON(200, session)
	})
	protected := g.Group("")
	protected.Use(a.authorize())
	a.postRoutes(protected, posts.NewRepository(db))
	protected.POST("/uploads", a.upload)
	protected.POST("/logout", func(c *gin.Context) {
		session := c.MustGet("adminSession").(admin.Session)
		if a.failed(c, a.sessions.Delete(c.Request.Context(), session.Hash)) {
			return
		}
		a.cookie(c, "", -1)
		c.JSON(200, admin.Session{})
	})
}

func (a *adminAPI) cookie(c *gin.Context, token string, age int) {
	http.SetCookie(c.Writer, &http.Cookie{Name: adminCookie, Value: token, Path: "/", MaxAge: age, HttpOnly: true, Secure: a.cfg.IsProduction(), SameSite: http.SameSiteLaxMode})
}
func (a *adminAPI) authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie(adminCookie)
		session, err := a.sessions.Get(c.Request.Context(), token)
		if a.failed(c, err) {
			return
		}
		if !session.Authenticated {
			writeError(c, 401, "admin_required", "Admin login required.")
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			if crossSiteRequest(c) || !admin.Equal(c.GetHeader("X-CSRF-Token"), session.CSRF) {
				writeError(c, 403, "invalid_csrf", "Invalid CSRF token or origin.")
				return
			}
		}
		c.Set("adminSession", session)
		c.Next()
	}
}
func (a *adminAPI) failed(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	a.logger.Error("admin request failed", "error", err)
	writeError(c, 500, "internal_error", "Internal server error.")
	return true
}
func decodeAdminJSON(c *gin.Context, limit int64, target any) bool {
	typ, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || typ != "application/json" {
		writeError(c, 415, "invalid_content_type", "Please send application/json.")
		return false
	}
	if limit <= 0 {
		limit = defaultCommentBodyLimit
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, limit))
	if err != nil || len(body) == 0 {
		writeError(c, 400, "invalid_body", "Request body is invalid or too large.")
		return false
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(body, &object) != nil || object == nil || json.Unmarshal(body, target) != nil {
		writeError(c, 400, "invalid_body", "Request body must be a JSON object.")
		return false
	}
	return true
}
