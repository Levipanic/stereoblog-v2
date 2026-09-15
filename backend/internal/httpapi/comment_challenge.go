package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/antispam"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
)

func commentChallengeHandler(service *antispam.Service, repository *posts.Repository, limiter *fixedWindowLimiter, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if allowed, retryAfter := limiter.allow(ClientIP(c)); !allowed {
			setRetryAfter(c, retryAfter)
			writeError(c, http.StatusTooManyRequests, "comment_rate_limited", "Too many comment requests. Please wait and try again.")
			return
		}
		postID, err := strconv.ParseInt(c.Param("post"), 10, 64)
		if err != nil || postID <= 0 {
			writeError(c, http.StatusBadRequest, "invalid_post_id", "Post ID must be a positive integer.")
			return
		}
		if _, err := repository.SlugByID(c.Request.Context(), postID); errors.Is(err, posts.ErrNotFound) {
			writeError(c, http.StatusNotFound, "post_not_found", "Post not found.")
			return
		} else if err != nil {
			logger.Error("comment challenge post lookup failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		challenge, err := service.IssueChallenge(postID)
		if err != nil {
			logger.Error("comment challenge generation failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		c.JSON(http.StatusOK, challenge)
	}
}
