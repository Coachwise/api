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
	return validateExerciseKind(form)
}

// validateExerciseKind checks the group half of the form. A group must say how
// it repeats (a round count or a time cap) and carry at least one exercise; a
// single exercise must carry none of that.
func validateExerciseKind(form *ExerciseForm) error {
	if form.Kind == "" || form.Kind == models.ExerciseKindSingle {
		if len(form.Items) > 0 {
			return errors.New("only a group exercise can contain items")
		}
		return nil
	}
	if form.Kind != models.ExerciseKindGroup {
		return errors.New("kind must be SINGLE or GROUP")
	}
	if len(form.Items) == 0 {
		return errors.New("a group must contain at least one exercise")
	}
	hasRounds := form.Rounds != nil && *form.Rounds > 0
	hasDuration := form.RoundDuration != nil && *form.RoundDuration > 0
	if hasRounds == hasDuration {
		return errors.New("a group needs either rounds or round_duration, not both")
	}
	if form.RoundRest < 0 {
		return errors.New("round_rest cannot be negative")
	}
	seen := make(map[uuid.UUID]bool, len(form.Items))
	for _, it := range form.Items {
		if it.RepCount != nil && it.Duration != nil {
			return errors.New("item cannot have both rep_count and duration")
		}
		if it.RepCount == nil && it.Duration == nil {
			return errors.New("item needs a rep_count or a duration")
		}
		if it.RepCount != nil && *it.RepCount < 0 {
			return errors.New("rep_count cannot be negative")
		}
		if it.Duration != nil && *it.Duration < 0 {
			return errors.New("duration cannot be negative")
		}
		if it.RestTime < 0 {
			return errors.New("rest_time cannot be negative")
		}
		if seen[it.ExerciseID] {
			return errors.New("an exercise can only appear once in a group")
		}
		seen[it.ExerciseID] = true
	}
	return nil
}

// resolveGroupItems turns the form's items into model rows, enforcing that each
// child is visible to the user and is itself SINGLE — that one rule is what
// keeps groups one level deep, so there is no cycle to detect.
func resolveGroupItems(form *ExerciseForm, userID uuid.UUID, selfID uuid.UUID) ([]models.ExerciseItem, error) {
	items := make([]models.ExerciseItem, 0, len(form.Items))
	for _, it := range form.Items {
		if it.ExerciseID == selfID {
			return nil, errors.New("a group cannot contain itself")
		}
		child, err := models.GetExrcise(it.ExerciseID)
		if err != nil {
			return nil, errors.New("exercise not found")
		}
		if !exerciseVisibleTo(child, userID) {
			return nil, errors.New("exercise not found")
		}
		if child.Kind == models.ExerciseKindGroup {
			return nil, errors.New("a group cannot contain another group")
		}
		items = append(items, models.ExerciseItem{
			ExerciseID: it.ExerciseID,
			RepCount:   it.RepCount,
			Duration:   it.Duration,
			RestTime:   it.RestTime,
		})
	}
	return items, nil
}

// applyExerciseKind copies the group half of the form onto the model, forcing a
// non-group back to a clean SINGLE so switching kind can't leave stale rounds.
func applyExerciseKind(ex *models.Exercise, form *ExerciseForm, items []models.ExerciseItem) {
	if form.Kind != models.ExerciseKindGroup {
		ex.Kind = models.ExerciseKindSingle
		ex.Rounds, ex.RoundDuration, ex.RoundRest, ex.Items = nil, nil, 0, nil
		return
	}
	ex.Kind = models.ExerciseKindGroup
	ex.Rounds = form.Rounds
	ex.RoundRest = form.RoundRest
	ex.RoundDuration = form.RoundDuration
	ex.Items = items
}

// applyExerciseMetrics copies the tracked-metric flags off the form, defaulting
// weight on and the rest off when a flag is omitted.
func applyExerciseMetrics(ex *models.Exercise, form *ExerciseForm) {
	ex.TrackWeight = form.TrackWeight == nil || *form.TrackWeight
	ex.TrackDistance = form.TrackDistance != nil && *form.TrackDistance
	ex.TrackGrade = form.TrackGrade != nil && *form.TrackGrade
	ex.TrackHeight = form.TrackHeight != nil && *form.TrackHeight
}

// exerciseVisibleTo reports whether a user may see an exercise: it is public,
// it is theirs, or it belongs to a plan assigned to them.
func exerciseVisibleTo(ex *models.Exercise, userID uuid.UUID) bool {
	if ex.Public {
		return true
	}
	if ex.UserID != nil && *ex.UserID == userID {
		return true
	}
	reachable, err := models.ExerciseReachableViaPlan(ex.ID, userID)
	return err == nil && reachable
}

// exerciseOwnedBy reports whether a user may edit or delete an exercise. Seeded
// library rows have user_id NULL and are editable by nobody through the API.
func exerciseOwnedBy(ex *models.Exercise, userID uuid.UUID) bool {
	return ex.UserID != nil && *ex.UserID == userID
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
		form := new(ExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		if form.Sets == nil {
			form.Sets = []SetForm{}
		}
		if err := validateExerciseForm(form); err != nil {
			AbortValidation(c, err)
			return
		}
		items, err := resolveGroupItems(form, user.ID, uuid.Nil)
		if err != nil {
			AbortValidation(c, err)
			return
		}
		ex := new(models.Exercise)
		utils.Copy(form, ex)
		ex.Name = html.EscapeString(ex.Name)
		ex.Description = html.EscapeString(ex.Description)
		ex.UserID = &user.ID
		// Anyone may create an exercise, but only for themselves — the public
		// library is curated via the seeder and admin panel, never the API.
		ex.Public = false
		applyExerciseMetrics(ex, form)
		applyExerciseKind(ex, form, items)
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
		// A group carries items instead of sets, so it's still a real creation.
		if len(ex.Sets) == 0 && len(ex.Items) == 0 {
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
		user := c.MustGet("user").(*models.User)
		if !exerciseVisibleTo(ex, user.ID) {
			AbortStatus(c, http.StatusNotFound, "exercise not found")
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
		if !exerciseOwnedBy(ex, user.ID) {
			AbortStatus(c, http.StatusForbidden, "forbidden")
			return
		}
		form := new(ExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		if form.Sets == nil {
			form.Sets = []SetForm{}
		}
		if err := validateExerciseForm(form); err != nil {
			AbortValidation(c, err)
			return
		}
		items, err := resolveGroupItems(form, user.ID, ex.ID)
		if err != nil {
			AbortValidation(c, err)
			return
		}
		utils.Copy(form, ex)
		ex.Name = html.EscapeString(ex.Name)
		ex.Description = html.EscapeString(ex.Description)
		applyExerciseMetrics(ex, form)
		applyExerciseKind(ex, form, items)
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
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx")
		page, _ := c.Get("paginate")
		items, total, err := models.ListExercisesPaginated(ctx.(context.Context), user.ID, publicFilter, search, category, sport, page.(database.Paginate))
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
		if !exerciseOwnedBy(existing, user.ID) {
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
