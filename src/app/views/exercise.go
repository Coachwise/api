package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/utils"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func exerciseGroup(router *gin.Engine) {
	g := router.Group("exercises")
	g.Use(auth.LoginRequired())

	g.POST("", func(c *gin.Context) {
		form := new(ExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ex := new(models.Exercise)
		utils.Copy(form, ex)
		user, _ := c.Get("user")
		ex.UserID = &user.(*models.User).ID
		for i := range ex.Sets {
			ex.Sets[i].SetNumber = i + 1
		}
		ctx, _ := c.Get("ctx")
		if err := ex.Create(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, ex)
	})

	g.GET("/:id", func(c *gin.Context) {
		ex, err := models.GetExrcise(uuid.MustParse(c.Param("id")))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ex)
	})

	g.PUT("/:id", func(c *gin.Context) {
		ex, err := models.GetExrcise(uuid.MustParse(c.Param("id")))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		form := new(ExerciseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		utils.Copy(form, ex)
		ctx, _ := c.Get("ctx")
		if err := ex.Update(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, ex)
	})

	/*
		 	g.DELETE("/:id", func(c *gin.Context) {
				ex, err := models.GetExrcise(uuid.MustParse(c.Param("id")))
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				ctx, _ := c.Get("ctx")
				if err := ex.Delete(ctx.(context.Context)); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "exercise deleted"})
			})

			g.GET("", func(c *gin.Context) {
				exs, err := models.GetExercises()
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, exs)
			})
	*/
}
