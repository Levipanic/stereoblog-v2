package httpapi

import (
	"errors"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
)

const likeRateLimitWindow = time.Minute

func postLikeHandler(repository *posts.Repository, cfg config.Likes, limiter *fixedWindowLimiter, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if crossSiteRequest(c) {
			writeError(c, http.StatusForbidden, "cross_site_request", "Cross-site write request rejected.")
			return
		}
		clientIP := ClientIP(c)
		allowed, retryAfter := limiter.allow(clientIP)
		if !allowed {
			setRetryAfter(c, retryAfter)
			writeError(c, http.StatusTooManyRequests, "like_rate_limited", "Too many like requests. Please wait and try again.")
			return
		}
		postID, err := strconv.ParseInt(c.Param("post"), 10, 64)
		if err != nil || postID <= 0 {
			writeError(c, http.StatusBadRequest, "invalid_post_id", "Post ID must be a positive integer.")
			return
		}
		result, err := repository.Like(c.Request.Context(), postID, posts.HashIP(cfg.IPHashSalt, clientIP), cfg.Cooldown, time.Now().UTC())
		if errors.Is(err, posts.ErrNotFound) {
			writeError(c, http.StatusNotFound, "post_not_found", "Post not found.")
			return
		}
		var cooldown *posts.CooldownError
		if errors.As(err, &cooldown) {
			setRetryAfter(c, cooldown.RetryAfter)
			writeError(c, http.StatusTooManyRequests, "like_cooldown", "You already liked this post recently.")
			return
		}
		if err != nil {
			logger.Error("post like failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func crossSiteRequest(c *gin.Context) bool {
	if strings.EqualFold(c.GetHeader("Sec-Fetch-Site"), "cross-site") {
		return true
	}
	location := c.GetHeader("Origin")
	if location == "" {
		location = c.GetHeader("Referer")
	}
	if location == "" {
		return false
	}
	parsed, err := url.Parse(location)
	return err != nil || parsed.Host == "" || !strings.EqualFold(parsed.Host, c.Request.Host)
}

func setRetryAfter(c *gin.Context, duration time.Duration) {
	seconds := max(1, int(math.Ceil(duration.Seconds())))
	c.Header("Retry-After", strconv.Itoa(seconds))
}
