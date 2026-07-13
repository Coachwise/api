package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/events"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"coachwise/src/database"
)

func connectionGroup(router *gin.Engine) {
	g := router.Group("connections")
	g.Use(auth.LoginRequired())

	// My established connections.
	g.GET("", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx")
		page, _ := c.Get("paginate")

		items, total, err := models.ListConnectionsPaginated(ctx.(context.Context), user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		if err := models.SetConnectionStatuses(ctx.(context.Context), user.ID, items); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	// Incoming requests addressed to me, filtered by status (default PENDING).
	g.GET("/requests", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx")
		page, _ := c.Get("paginate")

		status := c.Query("status")
		if status == "" {
			status = "PENDING"
		}

		items, total, err := models.ListIncomingRequestsPaginated(ctx.(context.Context), user.ID, status, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	g.POST("/requests/:id/accept", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid request id")
			return
		}
		ctx := c.MustGet("ctx")
		cr, err := models.AcceptConnectionRequest(ctx.(context.Context), id, user.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		events.EmitNotification(cr.RequesterID, &user.ID, models.NotifConnectionAccepted, sp("connection"), &cr.ID, nil)
		events.EmitSignal(cr.RequesterID, "connections") // their connection status changed
		c.JSON(http.StatusOK, cr)
	})

	g.POST("/requests/:id/reject", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid request id")
			return
		}
		ctx := c.MustGet("ctx")
		if err := models.RejectConnectionRequest(ctx.(context.Context), id, user.ID); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "rejected"})
	})
}
