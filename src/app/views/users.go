package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/events"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"coachwise/src/database"
)

// emptyToNil returns nil for a blank string so we store NULL rather than "".
func emptyToNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func userGroup(router *gin.Engine) {
	g := router.Group("users")
	g.Use(auth.LoginRequired())

	g.GET("", paginate(), func(c *gin.Context) {
		ctx := c.MustGet("ctx")
		user := c.MustGet("user").(*models.User)
		page, _ := c.Get("paginate")

		search := c.Query("search")
		coachOnly := c.Query("coach_only") == "true"
		// Optional sport filter (coaches only). Only the known enum values are
		// honoured; anything else is treated as "no filter".
		var sport *string
		if s := c.Query("sport"); s == "FITNESS" || s == "CLIMBING" || s == "THERAPEUTIC" {
			sport = &s
		}

		items, total, err := models.ListUsersPaginated(ctx.(context.Context), search, coachOnly, sport, user.ID, page.(database.Paginate))
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
		events.EmitNotification(id, &user.ID, models.NotifConnectionRequest, sp("connection_request"), &cr.ID, nil)
		events.EmitSignal(id, "connections") // refresh their incoming-requests list live
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
			AbortValidation(c, err)
			return
		}
		// Apply only the fields actually sent so omitted ones keep their value.
		if form.FirstName != nil {
			user.FirstName = form.FirstName
		}
		if form.LastName != nil {
			user.LastName = form.LastName
		}
		if form.JobTitle != nil {
			user.JobTitle = form.JobTitle
		}
		if form.Bio != nil {
			user.Bio = form.Bio
		}
		if form.Phone != nil {
			user.Phone = form.Phone
		}
		if form.AvatarID != nil {
			user.AvatarID = form.AvatarID
		}
		if form.Website != nil {
			user.Website = emptyToNil(*form.Website)
		}
		if form.Instagram != nil {
			// Store the bare handle (no leading @).
			user.Instagram = emptyToNil(strings.TrimPrefix(strings.TrimSpace(*form.Instagram), "@"))
		}
		if form.Birthday != nil {
			if strings.TrimSpace(*form.Birthday) == "" {
				user.Birthday = nil
			} else {
				d, err := time.Parse("2006-01-02", *form.Birthday)
				if err != nil {
					Abort(c, CodeValidation)
					return
				}
				user.Birthday = &d
			}
		}
		if form.Username != nil && *form.Username != user.Username {
			if _, err := models.GetUserByUsername(*form.Username); err == nil {
				Abort(c, CodeUsernameExists)
				return
			}
			user.Username = *form.Username
		}
		ctx := c.MustGet("ctx")
		if err := user.Update(ctx.(context.Context)); err != nil {
			AbortMsg(c, CodeUnknown, err.Error())
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
