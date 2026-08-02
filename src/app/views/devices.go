package views

import (
	"context"
	"net/http"
	"strings"

	"coachwise/src/app/auth"
	"coachwise/src/app/models"

	"github.com/gin-gonic/gin"
)

func deviceGroup(router *gin.Engine) {
	g := router.Group("devices")
	g.Use(auth.LoginRequired())

	// Register a push token. The app calls this on every launch, so it upserts.
	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var form DeviceForm
		if err := c.ShouldBindJSON(&form); err != nil {
			AbortValidation(c, err)
			return
		}
		platform := strings.ToLower(strings.TrimSpace(form.Platform))
		if !models.ValidPlatform(platform) {
			AbortStatus(c, http.StatusBadRequest, "invalid platform")
			return
		}

		ctx := c.MustGet("ctx").(context.Context)
		if err := models.RegisterDeviceToken(ctx, user.ID, strings.TrimSpace(form.Token), platform, form.Locale); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Unregister on logout, so a shared phone stops buzzing for this account.
	g.DELETE("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var form DeviceForm
		if err := c.ShouldBindJSON(&form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if err := models.UnregisterDeviceToken(ctx, strings.TrimSpace(form.Token), user.ID); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
}
