package views

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/events"
	"coachwise/src/database"
	"coachwise/src/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func supportGroup(router *gin.Engine) {
	g := router.Group("support")
	g.Use(auth.LoginRequired())

	// The user's tickets, most recently active first.
	g.GET("/tickets", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")

		items, total, err := models.ListUserTickets(ctx, user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	// Open a ticket with its first message.
	g.POST("/tickets", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)

		var form OpenTicketForm
		if err := c.ShouldBindJSON(&form); err != nil {
			AbortValidation(c, err)
			return
		}

		ticket, msg, err := models.OpenTicket(ctx, user.ID, form.Subject, form.Body)
		if err != nil {
			AbortServer(c, err)
			return
		}

		pingSupport(user, ticket, form.Body)
		c.JSON(http.StatusCreated, gin.H{"ticket": ticket, "message": msg})
	})

	// One ticket with its whole thread.
	g.GET("/tickets/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			Abort(c, CodeNotFound)
			return
		}
		ticket, err := models.GetTicket(ctx, id, user.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		if ticket == nil {
			Abort(c, CodeNotFound)
			return
		}
		msgs, err := models.ListTicketMessages(ctx, id)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"ticket": ticket, "messages": msgs})
	})

	// Add a reply. Only allowed when it is the user's turn.
	g.POST("/tickets/:id/messages", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			Abort(c, CodeNotFound)
			return
		}
		var form TicketMessageForm
		if err := c.ShouldBindJSON(&form); err != nil {
			AbortValidation(c, err)
			return
		}

		msg, err := models.AddUserMessage(ctx, id, user.ID, form.Body)
		if err != nil {
			switch {
			case errors.Is(err, models.ErrTicketNotFound):
				Abort(c, CodeNotFound)
			case errors.Is(err, models.ErrTicketClosed):
				Abort(c, CodeTicketClosed)
			case errors.Is(err, models.ErrNotYourTurn):
				Abort(c, CodeTicketNotYourTurn)
			default:
				AbortServer(c, err)
			}
			return
		}

		if ticket, _ := models.GetTicket(ctx, id, user.ID); ticket != nil {
			pingSupport(user, ticket, form.Body)
		}
		c.JSON(http.StatusCreated, gin.H{"message": msg})
	})

	// The user closes their own ticket.
	g.POST("/tickets/:id/close", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)

		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			Abort(c, CodeNotFound)
			return
		}
		ticket, err := models.CloseTicketByUser(ctx, id, user.ID)
		if err != nil {
			switch {
			case errors.Is(err, models.ErrTicketNotFound):
				Abort(c, CodeNotFound)
			case errors.Is(err, models.ErrTicketClosed):
				Abort(c, CodeTicketClosed)
			default:
				AbortServer(c, err)
			}
			return
		}
		// The user did this, so no notification to them — but their other devices
		// should reflect it, and Discord tells the team it's resolved.
		events.EmitSignal(user.ID, "support")
		events.EmitSupportPing(fmt.Sprintf("✅ **Support** — ticket `%s` closed by the user.", ticket.ID))
		c.JSON(http.StatusOK, gin.H{"ticket": ticket})
	})
}

// pingSupport queues a Discord heads-up that a ticket needs attention. Delivery
// is the worker's job (via NATS), so a slow webhook never shows up as latency on
// the user's request; the admin reads it and answers from the admin panel.
func pingSupport(user *models.User, ticket *models.SupportTicket, body string) {
	name := user.Username
	if user.FirstName != nil && *user.FirstName != "" {
		name = *user.FirstName
	}
	events.EmitSupportPing(fmt.Sprintf(
		"🎫 **Support** `#%s` — %s\n**%s**\n%s\n\n_Reply in the admin panel · ticket `%s`_",
		models.TicketRef(ticket.ID), name, ticket.Subject, utils.TruncateRunes(body, 500), ticket.ID,
	))
}
