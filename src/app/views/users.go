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

func userGroup(router *gin.Engine) {
	g := router.Group("users")
	g.Use(auth.LoginRequired())

	listHandler := func(c *gin.Context) {
		ctx := c.MustGet("ctx")
		username := ""
		if q, ok := c.GetQuery("username"); ok {
			username = q
		}
		users, err := models.ListUsers(ctx.(context.Context), username)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, users)
	}
	g.GET("/", listHandler)
	g.GET("", listHandler)

	g.GET("/:id", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}
		u, err := models.GetUser(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, u)
	})

	g.GET("/me", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		c.JSON(http.StatusOK, user)
	})

	g.GET("/me/plans", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")
		plans, err := models.ListUserAssignedPlans(ctx.(context.Context), user.ID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, plans)
	})

	g.PUT("/me", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		form := new(ProfileForm)
		if err := c.ShouldBindJSON(&form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		utils.Copy(form, user)
		ctx := c.MustGet("ctx")
		if err := user.Update(ctx.(context.Context)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, user)
	})

	g.DELETE("/me", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")
		if err := models.DeleteUser(ctx.(context.Context), user.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})
}
