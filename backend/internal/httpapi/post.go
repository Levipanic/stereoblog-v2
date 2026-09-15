package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
)

func postBySlugHandler(repository *posts.Repository, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := strings.TrimSpace(c.Param("post"))
		post, err := repository.BySlug(c.Request.Context(), slug)
		if errors.Is(err, posts.ErrNotFound) {
			writeError(c, http.StatusNotFound, "post_not_found", "Post not found.")
			return
		}
		if err != nil {
			logger.Error("post fetch failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		c.JSON(http.StatusOK, post)
	}
}

func postSlugByIDHandler(repository *posts.Repository, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || id <= 0 {
			writeError(c, http.StatusBadRequest, "invalid_post_id", "Post ID must be a positive integer.")
			return
		}
		result, err := repository.SlugByID(c.Request.Context(), id)
		if errors.Is(err, posts.ErrNotFound) {
			writeError(c, http.StatusNotFound, "post_not_found", "Post not found.")
			return
		}
		if err != nil {
			logger.Error("post ID resolution failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		c.JSON(http.StatusOK, result)
	}
}
