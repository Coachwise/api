package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"coachwise/src/database"
)

// sp returns a pointer to s (for optional notification entity_type/args).
func sp(s string) *string { return &s }

func notificationGroup(router *gin.Engine) {
	g := router.Group("notifications")
	g.Use(auth.LoginRequired())

	// The current user's notifications, newest first.
	g.GET("", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")
		items, total, err := models.ListNotificationsPaginated(ctx, user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	// Unread count for the bell badge.
	g.GET("/unread-count", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		n, err := models.CountUnreadNotifications(ctx, user.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"count": n})
	})

	// Mark all read.
	g.POST("/read-all", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		if err := models.MarkAllNotificationsRead(ctx, user.ID); err != nil {
			AbortServer(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// Mark one read.
	g.POST("/:id/read", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid notification id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if err := models.MarkNotificationRead(ctx, id, user.ID); err != nil {
			AbortServer(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
