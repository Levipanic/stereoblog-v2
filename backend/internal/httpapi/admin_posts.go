package httpapi

import (
	"errors"
	"strconv"

	"github.com/Levipanic/stereoblog-v2/backend/internal/content"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
	"github.com/gin-gonic/gin"
)

func (a *adminAPI) postRoutes(g *gin.RouterGroup, r *posts.Repository) {
	g.GET("/posts", feedHandler(r, a.logger))
	g.GET("/posts/:id", func(c *gin.Context) {
		id, ok := adminID(c)
		if !ok {
			return
		}
		p, err := r.AdminByID(c.Request.Context(), id)
		if !a.postError(c, err) {
			c.JSON(200, p)
		}
	})
	limiter := newFixedWindowLimiter(a.cfg.Admin.PostRateLimitWindow, a.cfg.Admin.PostRateLimitMax)
	save := func(create bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			var id int64
			if !create {
				var ok bool
				id, ok = adminID(c)
				if !ok {
					return
				}
			}
			if create {
				if ok, retry := limiter.allow(ClientIP(c)); !ok {
					setRetryAfter(c, retry)
					writeError(c, 429, "post_rate_limited", "Too many new posts.")
					return
				}
			}
			var input posts.WriteInput
			if !decodeAdminJSON(c, a.cfg.HTTP.JSONBodyLimit, &input) {
				return
			}
			limits := content.Limits{MaxBlocks: a.cfg.Posts.MaxBlocks, MaxText: a.cfg.Posts.MaxTextLength, MaxMediaText: a.cfg.Posts.MaxMediaTextLength}
			if err := input.Validate(limits); err != nil {
				writeError(c, 400, "invalid_post", err.Error())
				return
			}
			var err error
			if create {
				id, err = r.Create(c.Request.Context(), input)
			} else {
				err = r.Update(c.Request.Context(), id, input)
			}
			if a.postError(c, err) {
				return
			}
			p, err := r.AdminByID(c.Request.Context(), id)
			if a.postError(c, err) {
				return
			}
			status := 200
			if create {
				status = 201
			}
			c.JSON(status, p)
		}
	}
	g.POST("/posts", save(true))
	g.PUT("/posts/:id", save(false))
	g.DELETE("/posts/:id", func(c *gin.Context) {
		id, ok := adminID(c)
		if !ok {
			return
		}
		if !a.postError(c, r.Delete(c.Request.Context(), id)) {
			c.JSON(200, gin.H{"ok": true, "id": id})
		}
	})
}
func adminID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, 400, "invalid_id", "ID must be a positive integer.")
		return 0, false
	}
	return id, true
}
func (a *adminAPI) postError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, posts.ErrNotFound):
		writeError(c, 404, "post_not_found", "Post not found.")
	case errors.Is(err, posts.ErrSlugConflict):
		writeError(c, 409, "slug_conflict", "Slug already exists.")
	case errors.Is(err, posts.ErrSlugImmutable):
		writeError(c, 409, "slug_immutable", "Published slug cannot be changed.")
	default:
		return a.failed(c, err)
	}
	return true
}
