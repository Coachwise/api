package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/config"
	"coachwise/src/utils"
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	database "github.com/socious-io/pkg_database"
)

func coachGroup(router *gin.Engine) {
	g := router.Group("coaches")

	// Submit a coach application. Generates a unique application id + decision
	// token and pushes approve/reject capability links to Discord.
	g.POST("/apply", auth.LoginRequired(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		form := new(CoachApplicationForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		ctx := c.MustGet("ctx").(context.Context)
		app := &models.CoachApplication{
			UserID:          user.ID,
			FullName:        form.FullName,
			Specialty:       form.Specialty,
			ExperienceYears: form.ExperienceYears,
			Certifications:  form.Certifications,
			Bio:             form.Bio,
			Website:         form.Website,
			Instagram:       form.Instagram,
		}
		if err := app.Create(ctx); err != nil {
			AbortServer(c, err)
			return
		}

		notifyCoachApplication(app)
		c.JSON(http.StatusOK, app) // decision_token is json:"-", not exposed
	})

	// The current user's latest application (status), or null if none.
	g.GET("/application", auth.LoginRequired(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		app, err := models.LatestCoachApplication(user.ID)
		if err != nil {
			c.JSON(http.StatusOK, nil)
			return
		}
		c.JSON(http.StatusOK, app)
	})

	// Capability links — no auth; the decision token is the secret. GET so they
	// are clickable straight from Discord.
	g.GET("/applications/:id/approve", decisionHandler("APPROVED"))
	g.GET("/applications/:id/reject", decisionHandler("REJECTED"))

	// The coach's clients (their connections) enriched with the plans the coach
	// assigned to each. Powers the coach dashboard.
	g.GET("/clients", auth.LoginRequired(), paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")
		items, total, err := models.ListCoachClients(ctx, user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	// A coach's active packages (athlete-facing, e.g. tier comparison).
	g.GET("/:id/packages", auth.LoginRequired(), func(c *gin.Context) {
		coachID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid coach id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		items, err := models.ListCoachPackagesPublic(ctx, coachID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
	})
}

func decisionHandler(status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.String(http.StatusBadRequest, "invalid application id")
			return
		}
		token := c.Query("token")
		if token == "" {
			c.String(http.StatusBadRequest, "missing token")
			return
		}
		var note *string
		if n := c.Query("note"); n != "" {
			note = &n
		}

		ctx := c.MustGet("ctx").(context.Context)
		app, err := models.DecideCoachApplication(ctx, id, token, status, note)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		verb := "approved ✅"
		if status == "REJECTED" {
			verb = "rejected ❌"
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fmt.Sprintf(
			`<html><body style="font-family:sans-serif;text-align:center;padding:48px">`+
				`<h2>Application %s</h2><p>%s — %s</p></body></html>`,
			verb, app.FullName, app.Specialty)))
	}
}

// notifyCoachApplication posts the application summary + approve/reject links to Discord.
func notifyCoachApplication(app *models.CoachApplication) {
	base := config.Config.PublicURL
	approve := fmt.Sprintf("%s/coaches/applications/%s/approve?token=%s", base, app.ID, app.DecisionToken)
	reject := fmt.Sprintf("%s/coaches/applications/%s/reject?token=%s", base, app.ID, app.DecisionToken)
	// Angle brackets suppress Discord's link unfurling so the links aren't
	// auto-fetched (which would otherwise trigger a decision).
	msg := fmt.Sprintf(
		"**New coach application**\nName: %s\nSpecialty: %s\nExperience: %d yrs\nCertifications: %s\n\n✅ Approve: <%s>\n❌ Reject: <%s>",
		app.FullName, app.Specialty, app.ExperienceYears, app.Certifications, approve, reject,
	)
	utils.PostDiscordWebhook(config.Config.Discord.ApplicationWebhook, msg)
}
