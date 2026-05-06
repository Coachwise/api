package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func plansGroup(router *gin.Engine) {
	g := router.Group("plans")
	g.Use(auth.LoginRequired())

	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		form := new(CreatePlanForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Free users can only have 2 personal (non-public) plans
		if !user.Pro && !form.Public {
			ctx := c.MustGet("ctx")
			count, err := models.CountPersonalPlans(ctx.(context.Context), user.ID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if count >= 2 {
				c.JSON(http.StatusForbidden, gin.H{"error": "free users can create up to 2 personal plans"})
				return
			}
		}

		p := &models.Plan{
			UserID: user.ID,
			Public: form.Public,
			Name:   form.Name,
		}
		ctx := c.MustGet("ctx")
		if err := p.Create(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, p)
	})

	g.GET("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		onlyPublic := false
		if raw := c.Query("public"); raw != "" {
			val, _ := strconv.ParseBool(raw)
			onlyPublic = val
		}
		ctx := c.MustGet("ctx")
		plans, err := models.ListPlans(ctx.(context.Context), user.ID, onlyPublic)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Return pagination format for consistency with other list endpoints
		if plans == nil {
			plans = []models.Plan{}
		}
		c.JSON(http.StatusOK, gin.H{
			"items": plans,
			"total": len(plans),
		})
	})

	g.GET("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), id, user.ID)
		if err != nil {
			if err == models.ErrPlanAccessDenied {
				c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			}
			return
		}
		c.JSON(http.StatusOK, p)
	})

	g.PUT("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
			return
		}
		form := new(UpdatePlanForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), id, user.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		if p.UserID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can update plan"})
			return
		}
		if form.Name != nil {
			p.Name = *form.Name
		}
		if form.Public != nil {
			p.Public = *form.Public
		}
		if err := p.Update(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), id, user.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		if p.UserID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can delete plan"})
			return
		}
		if err := models.DeletePlan(ctx.(context.Context), id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	g.POST("/:id/exercises", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
			return
		}
		form := new(PlanExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		if p.UserID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can modify plan"})
			return
		}
		pe := &models.PlanExercise{
			PlanID:        planID,
			ExerciseID:    form.ExerciseID,
			ExerciseOrder: form.ExerciseOrder,
			RestTime:      form.RestTime,
			Intensity:     form.Intensity,
		}
		if err := models.AddPlanExercise(ctx.(context.Context), pe); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, pe)
	})

	g.GET("/:id/exercises", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
			return
		}
		ctx := c.MustGet("ctx")
		if _, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		items, err := models.ListPlanExercises(ctx.(context.Context), planID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, items)
	})

	g.DELETE("/:id/exercises/:exercise_id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
			return
		}
		exerciseID, err := uuid.Parse(c.Param("exercise_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exercise id"})
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		if p.UserID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can modify plan"})
			return
		}
		if err := models.RemovePlanExercise(ctx.(context.Context), planID, exerciseID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})

	g.POST("/:id/assign", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
			return
		}
		form := new(PlanAssignForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		if p.UserID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can assign plan"})
			return
		}
		assignee := &models.PlanAssignee{
			PlanID:   planID,
			UserID:   form.UserID,
			Assigner: user.ID,
		}
		if err := models.AssignPlan(ctx.(context.Context), assignee); err != nil {
			// Surface reason to clients and make debugging easier during development
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, assignee)
	})

	g.GET("/:id/assignments", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		if p.UserID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can view assignments"})
			return
		}
		list, err := models.ListPlanAssignees(ctx.(context.Context), planID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, list)
	})

	g.DELETE("/:id/assign/:user_id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
			return
		}
		userID, err := uuid.Parse(c.Param("user_id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
			return
		}
		if p.UserID != user.ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "only owner can unassign plan"})
			return
		}
		if err := models.UnassignPlan(ctx.(context.Context), planID, userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	})
}
