package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/utils"
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	database "github.com/socious-io/pkg_database"
)

func userGroup(router *gin.Engine) {
	g := router.Group("users")
	g.Use(auth.LoginRequired())

	g.GET("", paginate(), func(c *gin.Context) {
		ctx := c.MustGet("ctx")
		user := c.MustGet("user").(*models.User)
		page, _ := c.Get("paginate")

		search := c.Query("search")
		coachOnly := c.Query("coach_only") == "true"

		items, total, err := models.ListUsersPaginated(ctx.(context.Context), search, coachOnly, user.ID, page.(database.Paginate))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := models.SetConnectionStatuses(ctx.(context.Context), user.ID, items); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"items": items,
			"total": total,
		})
	})

	g.POST("/:id/connect", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}
		if id == user.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot connect to yourself"})
			return
		}
		ctx := c.MustGet("ctx")
		cr, err := models.SendConnectionRequest(ctx.(context.Context), user.ID, id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cr)
	})

	g.DELETE("/:id/connect", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}
		ctx := c.MustGet("ctx")
		if err := models.CancelConnectionRequest(ctx.(context.Context), user.ID, id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "canceled"})
	})

	g.GET("/:id", func(c *gin.Context) {
		viewer := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx")
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
		list := []models.User{*u}
		if err := models.SetConnectionStatuses(ctx.(context.Context), viewer.ID, list); err == nil {
			u = &list[0]
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
