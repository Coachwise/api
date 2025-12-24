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
)

func validateExerciseForm(form *ExerciseForm) error {
	if strings.TrimSpace(form.Name) == "" || strings.TrimSpace(form.Description) == "" {
		return errors.New("name and description are required")
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exercise id"})
	})

	g.POST("", func(c *gin.Context) {
		form := new(ExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ex := new(models.Exercise)
		utils.Copy(form, ex)
		ex.Name = html.EscapeString(ex.Name)
		ex.Description = html.EscapeString(ex.Description)
		user := c.MustGet("user")
		ex.UserID = &user.(*models.User).ID
		for i := range ex.Sets {
			ex.Sets[i].SetNumber = i + 1
			safeName := html.EscapeString(form.Sets[i].Name)
			ex.Sets[i].Name = utils.StrPtr(safeName)
		}
		ctx := c.MustGet("ctx")
		if err := ex.Create(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		status := http.StatusCreated
		if len(ex.Sets) == 0 {
			status = http.StatusOK
		}
		c.JSON(status, ex)
	})

	g.GET("/:id", func(c *gin.Context) {
		exID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exercise id"})
			return
		}
		ex, err := models.GetExrcise(exID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ex)
	})

	g.PUT("/:id", func(c *gin.Context) {
		exID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exercise id"})
			return
		}
		ex, err := models.GetExrcise(exID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user := c.MustGet("user").(*models.User)
		if ex.UserID != nil && ex.UserID.String() != user.ID.String() {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		form := new(ExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ex)
	})

	g.GET("", func(c *gin.Context) {
		var (
			publicFilter *bool
			nameFilter   string
		)
		if v, ok := c.GetQuery("public"); ok {
			val := strings.ToLower(v) == "true"
			publicFilter = &val
		}
		if v, ok := c.GetQuery("name"); ok {
			nameFilter = v
		}
		ctx := c.MustGet("ctx")
		exs, err := models.ListExercises(ctx.(context.Context), publicFilter, nameFilter)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, exs)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		exID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid exercise id"})
			return
		}
		user := c.MustGet("user").(*models.User)
		existing, err := models.GetExrcise(exID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if existing.UserID != nil && existing.UserID.String() != user.ID.String() {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		ctx := c.MustGet("ctx")
		if err := models.DeleteExercise(ctx.(context.Context), exID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

}
