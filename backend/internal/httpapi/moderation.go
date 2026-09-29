package httpapi

import (
	"errors"
	"github.com/Levipanic/stereoblog-v2/backend/internal/admin"
	"github.com/gin-gonic/gin"
)

func (a *adminAPI) moderationRoutes(g *gin.RouterGroup, m *admin.Moderation) {
	g.GET("/moderation", func(c *gin.Context) {
		result, err := m.Overview(c.Request.Context(), a.cfg.Comments.AdminListLimit)
		if !a.failed(c, err) {
			c.JSON(200, result)
		}
	})
	for _, action := range []string{"approve", "reject"} {
		g.POST("/comments/:id/"+action, func(c *gin.Context) {
			id, ok := adminID(c)
			if !ok {
				return
			}
			if a.moderationError(c, m.SetStatus(c.Request.Context(), id, action == "approve")) {
				return
			}
			status := "rejected"
			if action == "approve" {
				status = "visible"
			}
			c.JSON(200, gin.H{"ok": true, "id": id, "status": status})
		})
	}
	g.DELETE("/comments/:id", func(c *gin.Context) {
		id, ok := adminID(c)
		if !ok {
			return
		}
		postID, err := m.DeleteComment(c.Request.Context(), id)
		if !a.moderationError(c, err) {
			c.JSON(200, gin.H{"ok": true, "id": id, "post_id": postID})
		}
	})
	g.DELETE("/comment-mutes/:id", func(c *gin.Context) {
		id, ok := adminID(c)
		if !ok {
			return
		}
		if !a.moderationError(c, m.Unmute(c.Request.Context(), id)) {
			c.JSON(200, gin.H{"ok": true, "id": id})
		}
	})
}
func (a *adminAPI) moderationError(c *gin.Context, err error) bool {
	if errors.Is(err, admin.ErrNotFound) {
		writeError(c, 404, "moderation_not_found", "Comment or mute not found.")
		return true
	}
	return a.failed(c, err)
}
