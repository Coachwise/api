package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/events"
	"context"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"coachwise/src/database"
)

func plansGroup(router *gin.Engine) {
	g := router.Group("plans")
	g.Use(auth.LoginRequired())

	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		form := new(CreatePlanForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		// Plans created through the API are always personal. Free users are capped
		// at 2 plans, full stop — there is no way to make a plan public (which is
		// how a user previously both leaked a plan into everyone's list and dodged
		// this limit). A curated public library stays seeder/admin only.
		if !user.Pro {
			ctx := c.MustGet("ctx")
			count, err := models.CountPersonalPlans(ctx.(context.Context), user.ID)
			if err != nil {
				AbortServer(c, err)
				return
			}
			if count >= 2 {
				Abort(c, CodePlanLimit)
				return
			}
		}

		p := &models.Plan{
			UserID: user.ID,
			Public: false,
			Name:   form.Name,
		}
		ctx := c.MustGet("ctx")
		if err := p.Create(ctx.(context.Context)); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusCreated, p)
	})

	g.GET("", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		onlyPublic := false
		if raw := c.Query("public"); raw != "" {
			val, _ := strconv.ParseBool(raw)
			onlyPublic = val
		}
		ctx := c.MustGet("ctx")
		page, _ := c.Get("paginate")
		plans, total, err := models.ListPlansPaginated(ctx.(context.Context), user.ID, onlyPublic, c.Query("search"), page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": plans, "total": total})
	})

	g.GET("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid plan id")
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), id, user.ID)
		if err != nil {
			if err == models.ErrPlanAccessDenied {
				AbortStatus(c, http.StatusNotFound, "plan not found")
			} else {
				AbortServer(c, err)
			}
			return
		}
		c.JSON(http.StatusOK, p)
	})

	g.PUT("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid plan id")
			return
		}
		form := new(UpdatePlanForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), id, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "plan not found")
			return
		}
		if p.UserID != user.ID {
			Abort(c, CodeNotOwner)
			return
		}
		if form.Name != nil {
			p.Name = *form.Name
		}
		// `public` is intentionally not user-settable (see CreatePlanForm).
		if err := p.Update(ctx.(context.Context)); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, p)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid plan id")
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), id, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "plan not found")
			return
		}
		if p.UserID != user.ID {
			Abort(c, CodeNotOwner)
			return
		}
		if err := models.DeletePlan(ctx.(context.Context), id); err != nil {
			AbortServer(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	g.POST("/:id/exercises", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid plan id")
			return
		}
		form := new(PlanExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "plan not found")
			return
		}
		if p.UserID != user.ID {
			Abort(c, CodeNotOwner)
			return
		}
		if form.RestTime <= 0 {
			form.RestTime = time.Second
		}
		pe := &models.PlanExercise{
			PlanID:        planID,
			ExerciseID:    form.ExerciseID,
			ExerciseOrder: form.ExerciseOrder,
			RestTime:      form.RestTime,
			Intensity:     form.Intensity,
			Rounds:        form.Rounds,
			RoundRest:     form.RoundRest,
			RoundDuration: form.RoundDuration,
		}
		sets := make([]models.PlanExerciseSet, 0, len(form.Sets))
		for _, s := range form.Sets {
			set := models.PlanExerciseSet{
				RepCount: s.RepCount,
				Duration: s.Duration,
				RestTime: s.RestTime,
			}
			if s.Name != "" {
				name := html.EscapeString(s.Name)
				set.Name = &name
			}
			sets = append(sets, set)
		}
		if err := models.AddPlanExercise(ctx.(context.Context), pe, sets); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, pe)
	})

	g.GET("/:id/exercises", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid plan id")
			return
		}
		ctx := c.MustGet("ctx")
		if _, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID); err != nil {
			AbortStatus(c, http.StatusNotFound, "plan not found")
			return
		}
		items, err := models.ListPlanExercises(ctx.(context.Context), planID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, items)
	})

	g.DELETE("/:id/exercises/:exercise_id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid plan id")
			return
		}
		exerciseID, err := uuid.Parse(c.Param("exercise_id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid exercise id")
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "plan not found")
			return
		}
		if p.UserID != user.ID {
			Abort(c, CodeNotOwner)
			return
		}
		if err := models.RemovePlanExercise(ctx.(context.Context), planID, exerciseID); err != nil {
			AbortServer(c, err)
			return
		}
		c.Status(http.StatusOK)
	})

	g.POST("/:id/assign", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid plan id")
			return
		}
		form := new(PlanAssignForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "plan not found")
			return
		}
		if p.UserID != user.ID {
			Abort(c, CodeNotOwner)
			return
		}
		assignee := &models.PlanAssignee{
			PlanID:   planID,
			UserID:   form.UserID,
			Assigner: user.ID,
		}
		if err := models.AssignPlan(ctx.(context.Context), assignee); err != nil {
			// Surface reason to clients and make debugging easier during development
			AbortServer(c, err)
			return
		}
		events.EmitNotification(form.UserID, &user.ID, models.NotifPlanAssigned, sp("plan"), &planID, map[string]any{"name": p.Name})
		c.JSON(http.StatusOK, assignee)
	})

	g.GET("/:id/assignments", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid plan id")
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "plan not found")
			return
		}
		if p.UserID != user.ID {
			Abort(c, CodeNotOwner)
			return
		}
		list, err := models.ListPlanAssignees(ctx.(context.Context), planID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, list)
	})

	g.DELETE("/:id/assign/:user_id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		planID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid plan id")
			return
		}
		userID, err := uuid.Parse(c.Param("user_id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid user id")
			return
		}
		ctx := c.MustGet("ctx")
		p, err := models.GetPlanForUser(ctx.(context.Context), planID, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "plan not found")
			return
		}
		if p.UserID != user.ID {
			Abort(c, CodeNotOwner)
			return
		}
		if err := models.UnassignPlan(ctx.(context.Context), planID, userID); err != nil {
			AbortServer(c, err)
			return
		}
		events.EmitNotification(userID, &user.ID, models.NotifPlanRemoved, sp("plan"), &planID, map[string]any{"name": p.Name})
		c.Status(http.StatusOK)
	})
}
