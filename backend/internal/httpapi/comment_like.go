package httpapi

import (
	"errors"
	"github.com/Levipanic/stereoblog-v2/backend/internal/comments"
	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

func commentLikeHandler(repository *comments.Repository, cfg config.Likes, limiter *fixedWindowLimiter, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if crossSiteRequest(c) {
			writeError(c, http.StatusForbidden, "cross_site_request", "Cross-site write request rejected.")
			return
		}
		ip := ClientIP(c)
		if allowed, retry := limiter.allow(ip); !allowed {
			setRetryAfter(c, retry)
			writeError(c, 429, "like_rate_limited", "Too many like requests.")
			return
		}
		id, err := strconv.ParseInt(c.Param("comment"), 10, 64)
		if err != nil || id <= 0 {
			writeError(c, 400, "invalid_comment_id", "Comment ID must be a positive integer.")
			return
		}
		result, err := repository.Like(c.Request.Context(), id, posts.HashIP(cfg.IPHashSalt, ip), cfg.Cooldown, time.Now().UTC())
		var cooldown *posts.CooldownError
		switch {
		case errors.Is(err, comments.ErrNotFound):
			writeError(c, 404, "comment_not_found", "Comment not found.")
		case errors.As(err, &cooldown):
			setRetryAfter(c, cooldown.RetryAfter)
			writeError(c, 429, "like_cooldown", "You already liked this comment recently.")
		case err != nil:
			logger.Error("comment like failed", "error", err)
			writeError(c, 500, "internal_error", "Internal server error.")
		default:
			c.JSON(200, result)
		}
	}
}
