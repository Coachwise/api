package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/config"
	"coachwise/src/events"
	"coachwise/src/payments"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	database "github.com/socious-io/pkg_database"
)

// topUpReturnURL builds the frontend URL the gateway callback bounces the browser
// to, carrying the outcome so the app can toast + refresh the wallet.
func topUpReturnURL(result string) string {
	base := config.Config.Payments.ReturnURL
	if base == "" {
		base = "/"
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "topup=" + url.QueryEscape(result)
}

// walletGroup exposes the current user's wallet: derived balance, ledger,
// explicit top-up, and coach payout requests. The buy flows auto-top-up, so the
// explicit top-up endpoint is mostly for manual credit.
func walletGroup(router *gin.Engine) {
	g := router.Group("wallet", auth.LoginRequired(), withTimeout(10*time.Second))

	walletCurrency := func() string {
		if s, err := models.GetPlatformSettings(); err == nil {
			return s.DefaultCurrency
		}
		return "IRR"
	}

	// Balance (available + pending/escrow) for the user's default-currency wallet.
	g.GET("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		currency := walletCurrency()
		wallet, err := models.GetOrCreateUserWallet(user.ID, currency)
		if err != nil {
			AbortServer(c, err)
			return
		}
		balance, err := models.WalletBalanceOf(wallet.ID, currency)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"wallet_id": wallet.ID,
			"currency":  currency,
			"available": balance.Available,
			"pending":   balance.Pending,
		})
	})

	// Cumulative income (SALE credits): all-time total + current month. Distinct
	// from the spendable balance, which nets out payouts and escrow.
	g.GET("/income", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		currency := walletCurrency()
		wallet, err := models.GetOrCreateUserWallet(user.ID, currency)
		if err != nil {
			AbortServer(c, err)
			return
		}
		income, err := models.WalletIncomeOf(wallet.ID, currency)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"currency": currency, "total": income.Total, "month": income.Month})
	})

	// Ledger, newest first (?limit=&offset=).
	g.GET("/transactions", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		currency := walletCurrency()
		wallet, err := models.GetOrCreateUserWallet(user.ID, currency)
		if err != nil {
			AbortServer(c, err)
			return
		}
		limit, offset := parsePagination(c, 20, 100)
		txns, err := models.ListWalletTransactions(wallet.ID, limit, offset)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, txns)
	})

	// Manual top-up via the payment provider (stub auto-succeeds in beta).
	g.POST("/topup", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		form := new(TopUpForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		currencyIn := form.Currency
		if currencyIn == "" {
			currencyIn = walletCurrency()
		}
		currency, err := models.ValidateCurrency(currencyIn)
		if err != nil {
			Abort(c, CodeUnsupportedCurrency)
			return
		}
		res, err := payments.Charge(form.Provider, user.ID, form.Amount, currency)
		switch err {
		case payments.ErrNoProvider:
			Abort(c, CodeNoProvider)
			return
		case payments.ErrCurrencyUnsupported:
			Abort(c, CodeUnsupportedCurrency)
			return
		}
		if err != nil || res.Status != "PAID" {
			Abort(c, CodePaymentFailed)
			return
		}
		wallet, err := models.GetOrCreateUserWallet(user.ID, currency)
		if err != nil {
			AbortServer(c, err)
			return
		}
		if err := models.CreditTopUp(context.Background(), user.ID, wallet, form.Amount, res.Ref); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "topped up", "amount": form.Amount, "currency": currency})
	})

	// Start a redirect-gateway top-up (e.g. SEP). Returns a URL to send the
	// browser to; the gateway posts the result to the public callback below.
	g.POST("/topup/initiate", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		form := new(TopUpInitiateForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		pay, redirectURL, err := models.InitiateRedirectTopUp(ctx, user.ID, form.Amount, form.Provider, "")
		if err != nil {
			switch {
			case errors.Is(err, payments.ErrNoProvider), errors.Is(err, payments.ErrRedirectRequired):
				Abort(c, CodeNoProvider)
			case errors.Is(err, payments.ErrCurrencyUnsupported):
				Abort(c, CodeUnsupportedCurrency)
			case errors.Is(err, models.ErrInvalidAmount):
				Abort(c, CodeBadRequest)
			default:
				Abort(c, CodePaymentFailed)
			}
			return
		}
		c.JSON(http.StatusOK, gin.H{"payment_id": pay.ID, "redirect_url": redirectURL})
	})

	// Public gateway callback — the gateway (not our client) posts the result
	// here, so it lives OUTSIDE the auth group. It settles the top-up, then bounces
	// the user's browser back to the frontend with a status. Accepts POST or GET
	// (SEP's GetMethod variant).
	router.Any("/wallet/topup/callback", withTimeout(25*time.Second), func(c *gin.Context) {
		resNum := c.Request.FormValue("ResNum")
		refNum := c.Request.FormValue("RefNum")
		state := c.Request.FormValue("State")
		status := c.Request.FormValue("Status")
		stateOK := strings.EqualFold(state, "OK") || status == "2"

		result := "failed"
		if id, err := uuid.Parse(resNum); err == nil {
			ctx := c.MustGet("ctx").(context.Context)
			pay, credited, err := models.SettleRedirectTopUp(ctx, id, refNum, stateOK)
			switch {
			case credited && pay != nil:
				// Poke the buyer's live clients to refetch the wallet (and resume a
				// pending purchase) — no client polling needed.
				events.EmitSignal(pay.UserID, "wallet")
				result = "success"
			case err != nil && !errors.Is(err, models.ErrTopUpAmountMismatch):
				// Transient (e.g. gateway/verify) — payment stays PENDING for retry.
				result = "pending"
			}
		}
		c.Redirect(http.StatusFound, topUpReturnURL(result))
	})

	// Coach's payout destination for the wallet currency (or 404-ish empty).
	g.GET("/payout-account", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsCoach {
			Abort(c, CodeCoachOnly)
			return
		}
		acc, err := models.GetPayoutAccount(user.ID, walletCurrency())
		if err != nil {
			// Not set up yet — return an empty payload rather than an error so the
			// UI can render the "add your payout info" state cleanly.
			c.JSON(http.StatusOK, gin.H{"currency": walletCurrency(), "account": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"currency": walletCurrency(), "account": acc})
	})

	// Create/update the coach's payout destination. Method follows the currency.
	g.PUT("/payout-account", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsCoach {
			Abort(c, CodeCoachOnly)
			return
		}
		form := new(PayoutAccountForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		currency := walletCurrency()
		acc := &models.PayoutAccount{
			UserID:        user.ID,
			Currency:      currency,
			AccountHolder: form.AccountHolder,
		}
		switch currency {
		case "IRR":
			// Iran: a bank card number is enough. Accept spaces/dashes, store digits.
			digits := onlyDigits(deref(form.CardNumber))
			if len(digits) != 16 {
				Abort(c, CodeValidation)
				return
			}
			acc.Method = "CARD"
			acc.CardNumber = &digits
		default:
			// Other currencies need Stripe (Connect), which isn't wired yet.
			Abort(c, CodeNoProvider)
			return
		}
		saved, err := models.UpsertPayoutAccount(acc)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, saved)
	})

	// Coach requests a payout (top-out) against available balance.
	g.POST("/payout", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		if !user.IsCoach {
			Abort(c, CodeCoachOnly)
			return
		}
		form := new(PayoutForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		// A payout needs a destination to pay to.
		if _, err := models.GetPayoutAccount(user.ID, walletCurrency()); err != nil {
			Abort(c, CodePayoutAccountMissing)
			return
		}
		ctx := c.MustGet("ctx").(context.Context)
		payout, err := models.RequestPayout(ctx, user.ID, walletCurrency(), form.Amount, form.Note)
		if err != nil {
			payoutError(c, err)
			return
		}
		c.JSON(http.StatusOK, payout)
	})

	// Coach's payout history.
	g.GET("/payouts", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		page, _ := c.Get("paginate")
		items, total, err := models.ListPayoutsPaginated(user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})
}

// onlyDigits strips everything but ASCII digits (card numbers arrive with spaces
// or dashes). Persian/Arabic digits are normalized on the client before send.
func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func payoutError(c *gin.Context, err error) {
	switch err {
	case models.ErrPayoutExceeds:
		Abort(c, CodePayoutExceedsAvailable)
	case models.ErrInvalidAmount:
		Abort(c, CodeBadRequest)
	default:
		AbortServer(c, err)
	}
}
