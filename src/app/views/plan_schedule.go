package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	database "github.com/socious-io/pkg_database"
)

func planScheduleGroup(router *gin.Engine) {
	g := router.Group("schedules")
	g.Use(auth.LoginRequired())

	g.POST("", func(c *gin.Context) {
		form := new(PlanScheduleCreateForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		user := c.MustGet("user").(*models.User)

		scheduledFor, err := time.Parse("2006-01-02", form.ScheduledFor)
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid scheduled_for date")
			return
		}

		ctx := c.MustGet("ctx")

		ps := &models.PlanSchedule{
			UserID:       user.ID,
			PlanID:       form.PlanID,
			ScheduledFor: scheduledFor,
			Notes:        form.Notes,
		}

		if err := models.CreatePlanSchedule(ctx.(context.Context), ps); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusCreated, ps)
	})

	g.GET("", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx")
		page, _ := c.Get("paginate")

		items, total, err := models.ListPlanSchedulePaginated(ctx.(context.Context), user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"items": items,
			"total": total,
		})
	})

	g.PATCH("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid schedule id")
			return
		}
		form := new(PlanScheduleUpdateForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx")
		// Ensure ownership
		items, err := models.ListPlanSchedule(ctx.(context.Context), user.ID, time.Time{}, time.Now().AddDate(100, 0, 0))
		if err != nil {
			AbortServer(c, err)
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
			AbortStatus(c, http.StatusNotFound, "schedule not found")
			return
		}
		if form.Status != nil {
			target.Status = models.PlanScheduleStatus(*form.Status)
		}
		target.Notes = form.Notes
		if err := models.UpdatePlanSchedule(ctx.(context.Context), target); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, target)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid schedule id")
			return
		}
		ctx := c.MustGet("ctx")
		// Ensure ownership
		items, err := models.ListPlanSchedule(ctx.(context.Context), user.ID, time.Time{}, time.Now().AddDate(100, 0, 0))
		if err != nil {
			AbortServer(c, err)
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
			AbortStatus(c, http.StatusNotFound, "schedule not found")
			return
		}
		if err := models.DeletePlanSchedule(ctx.(context.Context), id); err != nil {
			AbortServer(c, err)
			return
		}
		c.Status(http.StatusOK)
	})
}
