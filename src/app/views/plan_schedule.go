package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func planScheduleGroup(router *gin.Engine) {
	g := router.Group("schedules")
	g.Use(auth.LoginRequired())

	g.POST("", func(c *gin.Context) {
		form := new(PlanScheduleCreateForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user := c.MustGet("user").(*models.User)

		scheduledFor, err := time.Parse("2006-01-02", form.ScheduledFor)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scheduled_for date"})
			return
		}

		ctx := c.MustGet("ctx")

		ps := &models.PlanSchedule{
			UserID:       user.ID,
			PlanID:       form.PlanID,
			ScheduledFor: scheduledFor,
		}
		if form.Status != nil {
			ps.Status = *form.Status
		}
		ps.Notes = form.Notes

		if err := models.CreatePlanSchedule(ctx.(context.Context), ps); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, ps)
	})

	g.GET("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		fromStr := c.Query("from")
		toStr := c.Query("to")
		if fromStr == "" || toStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from and to are required (YYYY-MM-DD)"})
			return
		}
		from, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date"})
			return
		}
		to, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date"})
			return
		}
		ctx := c.MustGet("ctx")
		items, err := models.ListPlanSchedule(ctx.(context.Context), user.ID, from, to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, items)
	})

	g.PATCH("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
			return
		}
		form := new(PlanScheduleUpdateForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx := c.MustGet("ctx")
		// Ensure ownership
		items, err := models.ListPlanSchedule(ctx.(context.Context), user.ID, time.Time{}, time.Now().AddDate(100, 0, 0))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var target *models.PlanSchedule
		for i := range items {
			if items[i].ID == id {
				target = &items[i]
				break
			}
		}
		if target == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}
		if form.Status != nil {
			target.Status = *form.Status
		}
		target.Notes = form.Notes
		if err := models.UpdatePlanSchedule(ctx.(context.Context), target); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, target)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
			return
		}
		ctx := c.MustGet("ctx")
		// Ensure ownership
		items, err := models.ListPlanSchedule(ctx.(context.Context), user.ID, time.Time{}, time.Now().AddDate(100, 0, 0))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		owned := false
		for _, item := range items {
			if item.ID == id {
				owned = true
				break
			}
		}
		if !owned {
			c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
			return
		}
		if err := models.DeletePlanSchedule(ctx.(context.Context), id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})
}
