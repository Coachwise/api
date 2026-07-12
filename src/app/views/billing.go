package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/payments"
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// withTimeout overrides the request's "ctx" with a longer deadline than the
// global 2s — the money flows do several ledger writes in one DB transaction.
func withTimeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Set("ctx", ctx)
		c.Next()
	}
}

// billingGroup exposes read-only pricing so the client can show the exact figure
// it will be charged. All money math is computed server-side (see models.Quote).
func billingGroup(router *gin.Engine) {
	g := router.Group("pricing", auth.LoginRequired())

	// Pro monthly price (default currency) + the shared duration tiers, so the FE
	// can render the Pro upsell without a round-trip per duration.
	g.GET("/pro", func(c *gin.Context) {
		settings, err := models.GetPlatformSettings()
		if err != nil {
			AbortServer(c, err)
			return
		}
		currency := c.DefaultQuery("currency", settings.DefaultCurrency)
		tiers, err := models.ListDurationTiers()
		if err != nil {
			AbortServer(c, err)
			return
		}
		monthly, err := models.QuotePro(currency, 1)
		if err != nil {
			billingError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"currency":       currency,
			"monthly_amount": monthly.UnitAmount,
			"tiers":          tiers,
		})
	})

	g.GET("/tiers", func(c *gin.Context) {
		tiers, err := models.ListDurationTiers()
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, tiers)
	})

	// Platform-supported currencies (enabled), for pickers.
	g.GET("/currencies", func(c *gin.Context) {
		currencies, err := models.ListCurrencies()
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, currencies)
	})

	// Payment providers that handle a currency (?currency=IRR) — for the Pro/top-up
	// provider picker (packages use /pricing/options which bundles providers).
	g.GET("/providers", func(c *gin.Context) {
		settings, err := models.GetPlatformSettings()
		if err != nil {
			AbortServer(c, err)
			return
		}
		currency := c.DefaultQuery("currency", settings.DefaultCurrency)
		c.JSON(http.StatusOK, payments.ProvidersFor(currency))
	})

	// Buy Pro for a duration. Auto-tops-up the wallet shortfall then pays, so the
	// athlete experiences a single "subscribe" action.
	b := router.Group("billing", auth.LoginRequired(), withTimeout(10*time.Second))
	b.POST("/pro", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		form := new(PurchaseForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		if form.Months <= 0 {
			Abort(c, CodeInvalidDuration)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		order, err := models.PurchasePro(ctx, user.ID, form.Currency, form.Provider, form.Months)
		if err != nil {
			purchaseError(c, err)
			return
		}
		c.JSON(http.StatusOK, order)
	})

	// Quote a purchase in a currency:
	//   ?kind=PRO&currency=IRR&months=3
	//   ?kind=PACKAGE&package_id=..&currency=IRR&months=3   (months ignored for one-time)
	g.GET("/quote", func(c *gin.Context) {
		settings, err := models.GetPlatformSettings()
		if err != nil {
			AbortServer(c, err)
			return
		}
		currency := c.DefaultQuery("currency", settings.DefaultCurrency)
		months, _ := strconv.Atoi(c.Query("months"))
		ctx := c.MustGet("ctx").(context.Context)

		switch c.Query("kind") {
		case "PRO":
			if months <= 0 {
				Abort(c, CodeInvalidDuration)
				return
			}
			quote, err := models.QuotePro(currency, months)
			if err != nil {
				billingError(c, err)
				return
			}
			c.JSON(http.StatusOK, quote)
		case "PACKAGE":
			pkgID, err := uuid.Parse(c.Query("package_id"))
			if err != nil {
				Abort(c, CodeBadRequest)
				return
			}
			pkg, err := models.GetPackage(ctx, pkgID)
			if err != nil {
				Abort(c, CodeNotFound)
				return
			}
			user := c.MustGet("user").(*models.User)
			quote, err := models.QuotePackage(pkg, currency, months, user.Pro)
			if err != nil {
				billingError(c, err)
				return
			}
			c.JSON(http.StatusOK, quote)
		default:
			Abort(c, CodeBadRequest)
		}
	})

	// Payment options for a package: the currencies it sells in that also have a
	// provider, each with its providers — drives the buyer's currency+provider UI.
	g.GET("/options", func(c *gin.Context) {
		if c.Query("kind") != "PACKAGE" {
			Abort(c, CodeBadRequest)
			return
		}
		pkgID, err := uuid.Parse(c.Query("package_id"))
		if err != nil {
			Abort(c, CodeBadRequest)
			return
		}
		opts, err := models.PackagePurchaseOptions(pkgID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, opts)
	})
}

// billingError maps the shared pricing/wallet model errors to error codes.
func billingError(c *gin.Context, err error) {
	switch err {
	case models.ErrInvalidDuration:
		Abort(c, CodeInvalidDuration)
	case models.ErrPriceNotConfigured:
		Abort(c, CodePriceNotConfigured)
	default:
		AbortServer(c, err)
	}
}

// purchaseError maps buy-flow errors (pricing + payment + business rules) to codes.
func purchaseError(c *gin.Context, err error) {
	switch err {
	case models.ErrInvalidDuration:
		Abort(c, CodeInvalidDuration)
	case models.ErrPriceNotConfigured:
		Abort(c, CodePriceNotConfigured)
	case models.ErrUnsupportedCurrency:
		Abort(c, CodeUnsupportedCurrency)
	case models.ErrPaymentFailed:
		Abort(c, CodePaymentFailed)
	case payments.ErrNoProvider:
		Abort(c, CodeNoProvider)
	case payments.ErrCurrencyUnsupported:
		Abort(c, CodeUnsupportedCurrency)
	case models.ErrPackageInactive:
		Abort(c, CodePackageUnavailable)
	case models.ErrSelfPurchase:
		Abort(c, CodeSelfAction)
	case models.ErrPackageNotFound:
		Abort(c, CodeNotFound)
	default:
		AbortServer(c, err)
	}
}
