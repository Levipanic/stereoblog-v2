package httpapi

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/comments"
	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
)

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewRouter(cfg config.Config, logger *slog.Logger, db *sql.DB) (*gin.Engine, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	router := gin.New()
	trustedProxies := []string(nil)
	if cfg.Server.TrustProxy {
		trustedProxies = cfg.Server.TrustedProxyCIDR
	}
	if err := router.SetTrustedProxies(trustedProxies); err != nil {
		return nil, err
	}

	router.HandleMethodNotAllowed = true
	router.Use(securityHeaders(), requestLogger(logger), recovery(logger))
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	postRepository := posts.NewRepository(db)
	commentRepository := comments.NewRepository(db)
	likeLimiter := newFixedWindowLimiter(likeRateLimitWindow, cfg.Likes.RateLimitMax)
	router.GET("/api/v1/posts", feedHandler(postRepository, logger))
	router.GET("/api/v1/posts/by-id/:id", postSlugByIDHandler(postRepository, logger))
	router.GET("/api/v1/posts/:post", postBySlugHandler(postRepository, logger))
	router.POST("/api/v1/posts/:id/likes", postLikeHandler(postRepository, cfg.Likes, likeLimiter, logger))
	router.GET("/api/v1/posts/:post/comments", commentsByPostHandler(commentRepository, logger))
	router.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "not_found", "Resource not found.")
	})
	router.NoMethod(func(c *gin.Context) {
		writeError(c, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	})

	return router, nil
}

func ClientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if address, err := netip.ParseAddr(ip); err == nil {
		return address.Unmap().String()
	}
	return ip
}

func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, errorResponse{Error: errorDetail{Code: code, Message: message}})
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "same-origin")
		c.Next()
	}
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		logger.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(started).Milliseconds(),
		)
	}
}

func recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "method", c.Request.Method, "path", c.Request.URL.Path)
				if c.Writer.Written() {
					c.Abort()
					return
				}
				writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			}
		}()
		c.Next()
	}
}
