package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/utils"
	"context"
	"errors"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"coachwise/src/database"
)

func validateExerciseForm(form *ExerciseForm) error {
	if strings.TrimSpace(form.Name) == "" || strings.TrimSpace(form.Description) == "" {
		return errors.New("name and description are required")
	}
	// The column is varchar(128); without this the overflow reaches Postgres and
	// comes back as a 500 — and a failed query counts against the circuit breaker.
	if len(form.Name) > 128 {
		return errors.New("name cannot be longer than 128 characters")
	}
	for _, set := range form.Sets {
		if set.RepCount != nil && set.Duration != nil {
			return errors.New("set cannot have both rep_count and duration")
		}
		if set.RepCount != nil && *set.RepCount < 0 {
			return errors.New("rep_count cannot be negative")
		}
		if set.Duration != nil && *set.Duration < 0 {
			return errors.New("duration cannot be negative")
		}
		if set.RestTime < 0 {
			return errors.New("rest_time cannot be negative")
		}
	}
	return nil
}

func exerciseGroup(router *gin.Engine) {
	g := router.Group("exercises")
	g.Use(auth.LoginRequired())

	// Treat trailing slash with no id as invalid identifier
	g.GET("/", func(c *gin.Context) {
		AbortStatus(c, http.StatusBadRequest, "invalid exercise id")
	})

	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsCoach {
			Abort(c, CodeCoachOnly)
			return
		}
		form := new(ExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		if form.Sets == nil {
			form.Sets = []struct {
				Name     string         `json:"name"`
				RestTime time.Duration  `json:"rest_time"`
				RepCount *int           `json:"rep_count"`
				Duration *time.Duration `json:"duration"`
			}{}
		}
		if err := validateExerciseForm(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ex := new(models.Exercise)
		utils.Copy(form, ex)
		ex.Name = html.EscapeString(ex.Name)
		ex.Description = html.EscapeString(ex.Description)
		ex.UserID = &user.ID
		for i := range ex.Sets {
			ex.Sets[i].SetNumber = i + 1
			safeName := html.EscapeString(form.Sets[i].Name)
			ex.Sets[i].Name = utils.StrPtr(safeName)
		}
		ctx := c.MustGet("ctx")
		if err := ex.Create(ctx.(context.Context)); err != nil {
			AbortServer(c, err)
			return
		}
		status := http.StatusCreated
		if len(ex.Sets) == 0 {
			status = http.StatusOK
		}
		c.JSON(status, ex)
	})

	g.GET("/categories", func(c *gin.Context) {
		cats, err := models.ListExerciseCategories(c.Query("sport"))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": cats})
	})

	g.GET("/:id", func(c *gin.Context) {
		exID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid exercise id")
			return
		}
		ex, err := models.GetExrcise(exID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, ex)
	})

	g.PUT("/:id", func(c *gin.Context) {
		exID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid exercise id")
			return
		}
		ex, err := models.GetExrcise(exID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		user := c.MustGet("user").(*models.User)
		if ex.UserID != nil && ex.UserID.String() != user.ID.String() {
			AbortStatus(c, http.StatusForbidden, "forbidden")
			return
		}
		form := new(ExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		if form.Sets == nil {
			form.Sets = []struct {
				Name     string         `json:"name"`
				RestTime time.Duration  `json:"rest_time"`
				RepCount *int           `json:"rep_count"`
				Duration *time.Duration `json:"duration"`
			}{}
		}
		if err := validateExerciseForm(form); err != nil {
			AbortValidation(c, err)
			return
		}
		utils.Copy(form, ex)
		ex.Name = html.EscapeString(ex.Name)
		ex.Description = html.EscapeString(ex.Description)
		for i := range ex.Sets {
			safeName := html.EscapeString(form.Sets[i].Name)
			ex.Sets[i].Name = utils.StrPtr(safeName)
		}
		ctx := c.MustGet("ctx")
		if err := ex.Update(ctx.(context.Context)); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, ex)
	})

	g.GET("", paginate(), func(c *gin.Context) {
		var publicFilter *bool
		if v, ok := c.GetQuery("public"); ok {
			val := strings.ToLower(v) == "true"
			publicFilter = &val
		}
		// Accept `search` (preferred) or the legacy `name` param.
		search := c.Query("search")
		if search == "" {
			search = c.Query("name")
		}
		category := c.Query("category") // exercise_categories.slug, or "" for all
		sport := c.Query("sport")       // exercise_sport_type, or "" for all
		ctx := c.MustGet("ctx")
		page, _ := c.Get("paginate")
		items, total, err := models.ListExercisesPaginated(ctx.(context.Context), publicFilter, search, category, sport, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	g.DELETE("/:id", func(c *gin.Context) {
		exID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid exercise id")
			return
		}
		user := c.MustGet("user").(*models.User)
		existing, err := models.GetExrcise(exID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		if existing.UserID != nil && existing.UserID.String() != user.ID.String() {
			AbortStatus(c, http.StatusForbidden, "forbidden")
			return
		}
		ctx := c.MustGet("ctx")
		if err := models.DeleteExercise(ctx.(context.Context), exID); err != nil {
			AbortServer(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

}
