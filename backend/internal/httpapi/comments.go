package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/comments"
)

func commentsByPostHandler(repository *comments.Repository, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		postID, err := strconv.ParseInt(c.Param("post"), 10, 64)
		if err != nil || postID <= 0 {
			writeError(c, http.StatusBadRequest, "invalid_post_id", "Post ID must be a positive integer.")
			return
		}
		items, err := repository.ByPostID(c.Request.Context(), postID)
		if errors.Is(err, comments.ErrPostNotFound) {
			writeError(c, http.StatusNotFound, "post_not_found", "Post not found.")
			return
		}
		if err != nil {
			logger.Error("comments fetch failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		c.JSON(http.StatusOK, items)
	}
}
