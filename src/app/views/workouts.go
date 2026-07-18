package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/utils"
	"context"
	"net/http"
	"time"

	"coachwise/src/database"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var allowedSessionTypes = map[string]bool{
	"PLAN":      true,
	"FREESTYLE": true,
	"STRENGTH":  true,
	"CLIMBING":  true,
}

func workoutsGroup(router *gin.Engine) {
	g := router.Group("workouts/sessions")
	g.Use(auth.LoginRequired())

	// Create session
	g.POST("", func(c *gin.Context) {
		form := new(CreateSessionForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		if !allowedSessionTypes[form.SessionType] {
			AbortStatus(c, http.StatusBadRequest, "invalid session type")
			return
		}

		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		session := &models.Session{
			UserID:      user.ID,
			SessionType: form.SessionType,
			PlanID:      form.PlanID,
			Status:      "ACTIVE",
			StartedAt:   time.Now(),
			Notes:       form.Notes,
		}

		if err := session.Create(ctx.(context.Context)); err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, session)
	})

	// Get all sessions for user
	g.GET("", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx")
		page, _ := c.Get("paginate")

		items, total, err := models.GetUserSessionsPaginated(ctx.(context.Context), user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}

		if items == nil {
			items = []models.Session{}
		}

		c.JSON(http.StatusOK, gin.H{
			"items": items,
			"total": total,
		})
	})

	// Get active sessions
	g.GET("/active", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		sessions, err := models.GetActiveSessions(ctx.(context.Context), user.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, sessions)
	})

	// Get session by ID
	g.GET("/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid session id")
			return
		}

		session, err := models.GetSession(id)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "session not found")
			return
		}

		c.JSON(http.StatusOK, session)
	})

	// Get daily analytics
	g.GET("/analytics/daily", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx")
		page, _ := c.Get("paginate")

		// Default: the caller's own analytics. A coach may view a client's via
		// ?athlete=<clientId> when that client is enrolled in one of their packages.
		target := user.ID
		if a := c.Query("athlete"); a != "" {
			clientID, err := uuid.Parse(a)
			if err != nil {
				AbortStatus(c, http.StatusBadRequest, "invalid athlete id")
				return
			}
			if ok, _ := models.IsCoachClient(ctx.(context.Context), user.ID, clientID); !ok {
				AbortStatus(c, http.StatusNotFound, "client not found")
				return
			}
			target = clientID
		}

		items, total, err := models.ListDailyAnalytics(ctx.(context.Context), target, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}

		if items == nil {
			items = []models.DailyAnalytics{}
		}

		c.JSON(http.StatusOK, gin.H{
			"items": items,
			"total": total,
		})
	})

	// Update session
	g.PUT("/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid session id")
			return
		}

		form := new(UpdateSessionForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		session, err := models.GetSession(id)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "session not found")
			return
		}

		user := c.MustGet("user").(*models.User)
		if session.UserID != user.ID {
			AbortStatus(c, http.StatusForbidden, "forbidden")
			return
		}

		ctx := c.MustGet("ctx")

		if form.Status != nil {
			session.Status = *form.Status
			if *form.Status == "COMPLETED" || *form.Status == "ABANDONED" {
				now := time.Now()
				session.EndedAt = &now
			}
		}
		if form.Notes != nil {
			session.Notes = form.Notes
		}

		session.Intensity = &form.Intensity
		session.Quality = &form.Quality

		if err := session.Update(ctx.(context.Context)); err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, session)
	})

	// Get session logs
	g.GET("/:id/logs", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid session id")
			return
		}

		ctx := c.MustGet("ctx")
		logs, err := models.GetSessionWorkoutLogs(ctx.(context.Context), id)
		if err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, logs)
	})
}

func workoutLogGroup(router *gin.Engine) {
	g := router.Group("workouts/logs")
	g.Use(auth.LoginRequired())

	// Create workout log
	g.POST("", func(c *gin.Context) {
		form := new(CreateWorkoutLogForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		ctx := c.MustGet("ctx")

		log := &models.WorkoutLog{}
		utils.Copy(form, log)
		if form.Completed != nil {
			log.Completed = *form.Completed
		}
		// Use created_at default; still stamp logged time for response consistency
		log.CreatedAt = time.Now()

		if err := log.Create(ctx.(context.Context)); err != nil {
			AbortServer(c, err)
			return
		}

		// Add tags if provided
		if len(form.Tags) > 0 {
			for _, tagName := range form.Tags {
				tag, err := models.GetOrCreateTag(ctx.(context.Context), tagName, false)
				if err != nil {
					continue
				}
				models.AddTagToWorkoutLog(ctx.(context.Context), log.ID, tag.ID)
			}
		}

		c.JSON(http.StatusOK, log)
	})

	// Update workout log
	g.PUT("/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid log id")
			return
		}

		form := new(UpdateWorkoutLogForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		log, err := models.GetWorkoutLog(id)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "log not found")
			return
		}
		if log.SessionID == nil {
			AbortStatus(c, http.StatusForbidden, "forbidden")
			return
		}

		session, err := models.GetSession(*log.SessionID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "session not found")
			return
		}

		user := c.MustGet("user").(*models.User)
		if session.UserID != user.ID {
			AbortStatus(c, http.StatusForbidden, "forbidden")
			return
		}

		ctx := c.MustGet("ctx")

		utils.Copy(form, log)

		if err := log.Update(ctx.(context.Context)); err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, log)
	})

	// Delete workout log
	g.DELETE("/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid log id")
			return
		}
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)

		if err := models.DeleteWorkoutLog(ctx, id, user.ID); err != nil {
			Abort(c, CodeNotFound)
			return
		}

		c.Status(http.StatusNoContent)
	})

	// Add tag to workout log
	g.POST("/:id/tags", func(c *gin.Context) {
		logID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid log id")
			return
		}

		var body struct {
			TagID string `json:"tag_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			AbortValidation(c, err)
			return
		}

		tagID, err := uuid.Parse(body.TagID)
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid tag id")
			return
		}

		ctx := c.MustGet("ctx")
		if err := models.AddTagToWorkoutLog(ctx.(context.Context), logID, tagID); err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
}

func tagGroup(router *gin.Engine) {
	g := router.Group("tags")
	g.Use(auth.LoginRequired())

	// Search tags
	g.GET("/search", func(c *gin.Context) {
		query := c.DefaultQuery("query", "")
		sportType := c.DefaultQuery("sport_type", "")
		limit := 20
		offset := 0

		ctx := c.MustGet("ctx")
		tags, totalCount, err := models.SearchTags(ctx.(context.Context), query, sportType, limit, offset)
		if err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"tags":        tags,
			"total_count": totalCount,
		})
	})
}
