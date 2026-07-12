package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/events"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
	database "github.com/socious-io/pkg_database"
)

// applyPackageForm copies form fields onto a package model, normalizing the
// optional/JSON fields (custom_features defaults to [], is_active to true).
func applyPackageForm(p *models.CoachPackage, form *PackageForm) {
	p.Name = form.Name
	p.Description = form.Description
	p.PriceMonthly = form.PriceMonthly
	p.PriceAnnual = form.PriceAnnual
	p.PriceOneTime = form.PriceOneTime
	p.TrialDays = form.TrialDays
	p.CheckInFrequency = form.CheckInFrequency
	p.VideoAccess = form.VideoAccess
	p.NutritionGuides = form.NutritionGuides
	p.Popular = form.Popular

	features := form.CustomFeatures
	if features == nil {
		features = []string{}
	}
	raw, _ := json.Marshal(features)
	p.CustomFeatures = types.JSONText(raw)

	p.IsActive = true
	if form.IsActive != nil {
		p.IsActive = *form.IsActive
	}
}

// coachOwnsPlans returns true when every plan id is owned by the coach.
func coachOwnsPlans(ctx context.Context, coachID uuid.UUID, planIDs []uuid.UUID) bool {
	for _, id := range planIDs {
		p, err := models.GetPlanForUser(ctx, id, coachID)
		if err != nil || p.UserID != coachID {
			return false
		}
	}
	return true
}

func packagesGroup(router *gin.Engine) {
	g := router.Group("packages")
	g.Use(auth.LoginRequired())

	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsCoach {
			Abort(c, CodeCoachOnly)
			return
		}
		form := new(PackageForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if len(form.PlanIDs) > 0 && !coachOwnsPlans(ctx, user.ID, form.PlanIDs) {
			AbortStatus(c, http.StatusBadRequest, "all bundled plans must be owned by the coach")
			return
		}
		p := &models.CoachPackage{CoachID: user.ID}
		applyPackageForm(p, form)
		if err := p.Create(ctx); err != nil {
			AbortServer(c, err)
			return
		}
		if err := models.SetPackagePlans(ctx, p.ID, form.PlanIDs); err != nil {
			AbortServer(c, err)
			return
		}
		fresh, err := models.GetPackage(ctx, p.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusCreated, fresh)
	})

	g.GET("", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")
		items, total, err := models.ListCoachPackagesPaginated(ctx, user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	// The current user's active package subscriptions (athlete side).
	g.GET("/subscriptions", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")
		subs, total, err := models.ListClientSubscriptionsPaginated(ctx, user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": subs, "total": total})
	})

	g.GET("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid package id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		p, err := models.GetPackageForCoach(ctx, id, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "package not found")
			return
		}
		c.JSON(http.StatusOK, p)
	})

	g.PUT("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid package id")
			return
		}
		form := new(PackageForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		p, err := models.GetPackageForCoach(ctx, id, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "package not found")
			return
		}
		if len(form.PlanIDs) > 0 && !coachOwnsPlans(ctx, user.ID, form.PlanIDs) {
			AbortStatus(c, http.StatusBadRequest, "all bundled plans must be owned by the coach")
			return
		}
		applyPackageForm(p, form)
		if err := p.Update(ctx); err != nil {
			AbortServer(c, err)
			return
		}
		if form.PlanIDs != nil {
			if err := models.SetPackagePlans(ctx, p.ID, form.PlanIDs); err != nil {
				AbortServer(c, err)
				return
			}
		}
		fresh, err := models.GetPackage(ctx, p.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, fresh)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid package id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if _, err := models.GetPackageForCoach(ctx, id, user.ID); err != nil {
			AbortStatus(c, http.StatusNotFound, "package not found")
			return
		}
		if err := models.DeletePackage(ctx, id); err != nil {
			AbortServer(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	g.PUT("/:id/plans", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid package id")
			return
		}
		form := new(SetPackagePlansForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if _, err := models.GetPackageForCoach(ctx, id, user.ID); err != nil {
			AbortStatus(c, http.StatusNotFound, "package not found")
			return
		}
		if len(form.PlanIDs) > 0 && !coachOwnsPlans(ctx, user.ID, form.PlanIDs) {
			AbortStatus(c, http.StatusBadRequest, "all bundled plans must be owned by the coach")
			return
		}
		if err := models.SetPackagePlans(ctx, id, form.PlanIDs); err != nil {
			AbortServer(c, err)
			return
		}
		fresh, err := models.GetPackage(ctx, id)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, fresh)
	})

	// Per-currency prices for a package. Public GET (buyers see options); coach-only
	// PUT/DELETE to manage them (the package builder).
	g.GET("/:id/prices", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			Abort(c, CodeBadRequest)
			return
		}
		prices, err := models.ListPackagePrices(id)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, prices)
	})

	g.PUT("/:id/prices", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			Abort(c, CodeBadRequest)
			return
		}
		form := new(PackagePriceForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		currency, err := models.ValidateCurrency(form.Currency)
		if err != nil {
			Abort(c, CodeUnsupportedCurrency)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if _, err := models.GetPackageForCoach(ctx, id, user.ID); err != nil {
			Abort(c, CodeNotOwner)
			return
		}
		if err := models.SetPackagePrice(ctx, id, currency, form.Amount); err != nil {
			AbortServer(c, err)
			return
		}
		prices, _ := models.ListPackagePrices(id)
		c.JSON(http.StatusOK, prices)
	})

	g.DELETE("/:id/prices/:currency", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			Abort(c, CodeBadRequest)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		if _, err := models.GetPackageForCoach(ctx, id, user.ID); err != nil {
			Abort(c, CodeNotOwner)
			return
		}
		if err := models.DeletePackagePrice(ctx, id, c.Param("currency")); err != nil {
			AbortServer(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	})

	g.POST("/:id/assign", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid package id")
			return
		}
		form := new(PlanAssignForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		pkg, err := models.GetPackageForCoach(ctx, id, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "package not found")
			return
		}
		// Enroll the user as a client (creates the subscription) and assign the
		// package's bundled plans to them.
		sub, err := models.EnrollClient(ctx, id, user.ID, form.UserID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		events.EmitNotification(form.UserID, &user.ID, models.NotifPackageAssigned, sp("package"), &id, map[string]any{"name": pkg.Name})
		c.JSON(http.StatusOK, sub)
	})

	// Coach removes a client's subscription to the package and unassigns the
	// plans that came from it (manually-assigned plans are kept).
	g.DELETE("/:id/subscribers/:user_id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid package id")
			return
		}
		clientID, err := uuid.Parse(c.Param("user_id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid user id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		pkg, err := models.GetPackageForCoach(ctx, id, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "package not found")
			return
		}
		if err := models.UnsubscribeClient(ctx, id, clientID); err != nil {
			AbortServer(c, err)
			return
		}
		events.EmitNotification(clientID, &user.ID, models.NotifPackageRemoved, sp("package"), &id, map[string]any{"name": pkg.Name})
		c.Status(http.StatusOK)
	})

	// Athlete-driven enrollment: the current user subscribes to an active package.
	g.POST("/:id/subscribe", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid package id")
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		pkg, err := models.GetPackage(ctx, id)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "package not found")
			return
		}
		if !pkg.IsActive {
			Abort(c, CodePackageUnavailable)
			return
		}
		if pkg.CoachID == user.ID {
			Abort(c, CodeSelfAction)
			return
		}
		sub, err := models.EnrollClient(ctx, id, pkg.CoachID, user.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		events.EmitNotification(pkg.CoachID, &user.ID, models.NotifPackageSubscribed, sp("package"), &id, map[string]any{"name": pkg.Name})
		c.JSON(http.StatusOK, sub)
	})

	// Athlete buys the package for a duration: pays (auto top-up), credits the
	// coach (escrow, net of fee), enrolls + assigns plans, and grants Pro.
	g.POST("/:id/purchase", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			Abort(c, CodeBadRequest)
			return
		}
		form := new(PurchaseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		// The purchase does several ledger writes in one tx — allow more than 2s.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		order, err := models.PurchasePackage(ctx, user.ID, id, form.Currency, form.Provider, form.Months, user.Pro)
		if err != nil {
			purchaseError(c, err)
			return
		}
		if order.CoachID != nil {
			if pkg, perr := models.GetPackage(ctx, id); perr == nil {
				events.EmitNotification(*order.CoachID, &user.ID, models.NotifPackageSubscribed, sp("package"), &id, map[string]any{"name": pkg.Name})
			}
		}
		c.JSON(http.StatusOK, order)
	})
}
