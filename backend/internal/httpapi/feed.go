package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
)

func feedHandler(repository *posts.Repository, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := posts.DefaultLimit
		if raw := c.Query("limit"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > posts.MaxLimit {
				writeError(c, http.StatusBadRequest, "invalid_limit", "Limit must be between 1 and 50.")
				return
			}
			limit = value
		}
		page, err := repository.Feed(c.Request.Context(), c.Query("cursor"), limit)
		if errors.Is(err, posts.ErrInvalidCursor) {
			writeError(c, http.StatusBadRequest, "invalid_cursor", "Cursor is invalid.")
			return
		}
		if err != nil {
			logger.Error("feed failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		c.JSON(http.StatusOK, page)
	}
}
