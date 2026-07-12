package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/events"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	database "github.com/socious-io/pkg_database"
)

func testsGroup(router *gin.Engine) {
	g := router.Group("tests")
	g.Use(auth.LoginRequired())

	// --- Coach: test templates ---

	// Any user can build a test: coaches use them as templates to assign, athletes
	// as personal protocols to re-run on themselves.
	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		form := new(TestForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		t := &models.Test{CoachID: user.ID, Name: form.Name, Description: form.Description, Public: form.Public}
		if err := t.Create(ctx); err != nil {
			AbortServer(c, err)
			return
		}
		if err := models.SetTestItems(ctx, t.ID, form.Items); err != nil {
			AbortServer(c, err)
			return
		}
		fresh, err := models.GetTest(ctx, t.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusCreated, fresh)
	})

	g.GET("", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")
		items, total, err := models.ListCoachTestsPaginated(ctx, user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	// The coach's assigned protocols per client (client + run stats).
	g.GET("/assignments", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		items, err := models.ListCoachAssignments(ctx, user.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
	})

	// Protocols a coach has assigned to the current athlete (runnable like own).
	g.GET("/assigned", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")
		items, total, err := models.ListAssignedTestsPaginated(ctx, user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	// --- Requests (must be registered before /:id) ---

	// Coach's sent requests (for review).
	g.GET("/requests", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")
		items, total, err := models.ListCoachTestRequests(ctx, user.ID, c.Query("status"), page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	// The current athlete's assigned tests.
	g.GET("/requests/assigned", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")
		items, total, err := models.ListAthleteTestRequests(ctx, user.ID, c.Query("status"), page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	g.GET("/requests/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid request id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		tr, err := models.GetTestRequest(ctx, id)
		owns := err == nil && (tr.AthleteID == user.ID || (tr.CoachID != nil && *tr.CoachID == user.ID))
		if !owns {
			AbortStatus(c, http.StatusNotFound, "test request not found")
			return
		}
		c.JSON(http.StatusOK, tr)
	})

	// Athlete submits results.
	g.POST("/requests/:id/submit", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid request id")
			return
		}
		form := new(TestSubmitForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if err := models.SubmitTestRequest(ctx, id, user.ID, form.Records); err != nil {
			AbortServer(c, err)
			return
		}
		fresh, err := models.GetTestRequest(ctx, id)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, fresh)
	})

	// Athlete records their own assessment (no coach, no template).
	g.POST("/requests/self", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		form := new(SelfAssessmentForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		tr, err := models.CreateSelfAssessment(ctx, user.ID, form.Name, form.Records)
		if err != nil {
			AbortServer(c, err)
			return
		}
		fresh, err := models.GetTestRequest(ctx, tr.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusCreated, fresh)
	})

	// Coach acknowledges (marks seen) a submitted assessment.
	g.POST("/requests/:id/seen", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsCoach {
			Abort(c, CodeCoachOnly)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid request id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if err := models.MarkTestRequestSeen(ctx, id, user.ID); err != nil {
			AbortServer(c, err)
			return
		}
		fresh, err := models.GetTestRequest(ctx, id)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, fresh)
	})

	// --- Coach: single test by id ---

	g.GET("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid test id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		// The owner or an athlete it's been assigned to may view the protocol.
		if ok, _ := models.CanRunTest(ctx, id, user.ID); !ok {
			AbortStatus(c, http.StatusNotFound, "test not found")
			return
		}
		t, err := models.GetTest(ctx, id)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "test not found")
			return
		}
		c.JSON(http.StatusOK, t)
	})

	g.PUT("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid test id")
			return
		}
		form := new(TestForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		t, err := models.GetTestForCoach(ctx, id, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "test not found")
			return
		}
		t.Name = form.Name
		t.Description = form.Description
		t.Public = form.Public
		if err := t.Update(ctx); err != nil {
			AbortServer(c, err)
			return
		}
		if form.Items != nil {
			if err := models.SetTestItems(ctx, t.ID, form.Items); err != nil {
				AbortServer(c, err)
				return
			}
		}
		fresh, err := models.GetTest(ctx, t.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, fresh)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid test id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if _, err := models.GetTestForCoach(ctx, id, user.ID); err != nil {
			AbortStatus(c, http.StatusNotFound, "test not found")
			return
		}
		if err := models.DeleteTest(ctx, id); err != nil {
			AbortServer(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// Coach requests an athlete to take a test.
	g.POST("/:id/request", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid test id")
			return
		}
		form := new(TestRequestForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		t, err := models.GetTestForCoach(ctx, id, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "test not found")
			return
		}
		tr := &models.TestRequest{TestID: &id, CoachID: &user.ID, AthleteID: form.AthleteID, Note: form.Note}
		if err := models.CreateTestRequest(ctx, tr); err != nil {
			AbortServer(c, err)
			return
		}
		events.EmitNotification(form.AthleteID, &user.ID, models.NotifAssessmentAssigned, sp("test"), &id, map[string]any{"name": t.Name})
		c.JSON(http.StatusCreated, tr)
	})

	// The owner runs their own protocol, recording a new dated run of results.
	g.POST("/:id/run", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid test id")
			return
		}
		form := new(TestSubmitForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if ok, _ := models.CanRunTest(ctx, id, user.ID); !ok {
			AbortStatus(c, http.StatusNotFound, "protocol not found")
			return
		}
		tr, err := models.RunProtocol(ctx, id, user.ID, form.Records)
		if err != nil {
			AbortServer(c, err)
			return
		}
		// Notify the owning coach when an athlete runs a protocol they assigned.
		if t, e := models.GetTest(ctx, id); e == nil && t.CoachID != user.ID {
			events.EmitNotification(t.CoachID, &user.ID, models.NotifAssessmentSubmitted, sp("test"), &id, map[string]any{"name": t.Name})
		}
		fresh, err := models.GetTestRequest(ctx, tr.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusCreated, fresh)
	})

	// The owner's dated run history for a protocol (with per-run records).
	g.GET("/:id/runs", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid test id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		// Default: the caller's own runs. A coach may view a client's runs of a
		// protocol they own via ?athlete=<clientId>.
		target := user.ID
		if a := c.Query("athlete"); a != "" {
			clientID, err := uuid.Parse(a)
			if err != nil {
				AbortStatus(c, http.StatusBadRequest, "invalid athlete id")
				return
			}
			if _, err := models.GetTestForCoach(ctx, id, user.ID); err != nil {
				AbortStatus(c, http.StatusNotFound, "protocol not found")
				return
			}
			if ok, _ := models.IsTestAssignedTo(ctx, id, clientID); !ok {
				AbortStatus(c, http.StatusNotFound, "client not assigned")
				return
			}
			target = clientID
		} else if ok, _ := models.CanRunTest(ctx, id, user.ID); !ok {
			AbortStatus(c, http.StatusNotFound, "protocol not found")
			return
		}
		page, _ := c.Get("paginate")
		runs, total, err := models.ListProtocolRunsPaginated(ctx, id, target, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": runs, "total": total})
	})
}

func achievementGroup(router *gin.Engine) {
	g := router.Group("achievements")
	g.Use(auth.LoginRequired())

	// Coach grants a badge to an athlete.
	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsCoach {
			Abort(c, CodeCoachOnly)
			return
		}
		form := new(AchievementForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		a := &models.Achievement{AthleteID: form.AthleteID, CoachID: user.ID, Title: form.Title, Description: form.Description}
		if err := models.GrantAchievement(ctx, a); err != nil {
			AbortServer(c, err)
			return
		}
		events.EmitNotification(a.AthleteID, &user.ID, models.NotifBadgeGranted, sp("achievement"), &a.ID, map[string]any{"title": a.Title})
		c.JSON(http.StatusCreated, a)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid achievement id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if err := models.DeleteAchievement(ctx, id, user.ID); err != nil {
			AbortStatus(c, http.StatusNotFound, "achievement not found")
			return
		}
		c.Status(http.StatusNoContent)
	})

	// The current user curates their own profile trophy case (order + hidden).
	g.PUT("/layout", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		form := new(AchievementLayoutForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		layout := models.AchievementLayout{Order: form.Order, Hidden: form.Hidden}
		if err := models.SetAchievementLayout(ctx, user.ID, layout); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, layout)
	})

	// A user's profile achievements: derived PRs + coach-granted badges, plus the
	// owner's curated layout and (for coaches) their active-client count. Hidden
	// items are stripped for anyone other than the profile owner.
	router.GET("/users/:id/achievements", auth.LoginRequired(), func(c *gin.Context) {
		viewer := c.MustGet("user").(*models.User)
		userID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid user id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		prs, err := models.ListPersonalRecords(ctx, userID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		badges, err := models.ListAchievements(ctx, userID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		layout, err := models.GetAchievementLayout(ctx, userID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		activeClients, err := models.CountActiveClients(ctx, userID)
		if err != nil {
			AbortServer(c, err)
			return
		}

		// Non-owners never see hidden items (nor the hidden list itself).
		if viewer.ID != userID {
			hidden := make(map[string]bool, len(layout.Hidden))
			for _, k := range layout.Hidden {
				hidden[k] = true
			}
			visiblePRs := make([]models.PersonalRecord, 0, len(prs))
			for _, r := range prs {
				if !hidden["record:"+r.ExerciseID.String()] {
					visiblePRs = append(visiblePRs, r)
				}
			}
			prs = visiblePRs
			visibleBadges := make([]models.Achievement, 0, len(badges))
			for _, b := range badges {
				if !hidden["badge:"+b.ID.String()] {
					visibleBadges = append(visibleBadges, b)
				}
			}
			badges = visibleBadges
			layout.Hidden = []string{}
		}

		c.JSON(http.StatusOK, gin.H{
			"records":        prs,
			"badges":         badges,
			"layout":         layout,
			"active_clients": activeClients,
		})
	})
}
