package views

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/database"
	"coachwise/src/events"
	"coachwise/src/llm"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// requireAI aborts with CodeAIDisabled when no model is configured, so the
// client localizes the message instead of the BE authoring one.
func requireAI(c *gin.Context) bool {
	if !llm.Enabled() {
		Abort(c, CodeAIDisabled)
		return false
	}
	return true
}

// storedAction mirrors the persisted proposal shape (see events/ai.go) for
// recording client-executed write results.
type storedAction struct {
	Name   string          `json:"name"`
	Args   json.RawMessage `json:"args"`
	Kind   string          `json:"kind"`
	Status string          `json:"status"`
	Result json.RawMessage `json:"result,omitempty"`
}

func aiGroup(router *gin.Engine) {
	g := router.Group("ai")
	g.Use(auth.LoginRequired())

	// Start a conversation, optionally with a first message.
	g.POST("/conversations", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		var form AIStartForm
		if err := c.ShouldBindJSON(&form); err != nil {
			AbortValidation(c, err)
			return
		}
		if form.Text != "" && !requireAI(c) {
			return
		}
		conv, err := models.CreateConversation(ctx, user.ID, form.Title)
		if err != nil {
			AbortServer(c, err)
			return
		}
		var pending *models.AIMessage
		if form.Text != "" {
			pending, err = enqueueTurn(ctx, conv.ID, user.ID, form.Text)
			if err != nil {
				AbortServer(c, err)
				return
			}
		}
		c.JSON(http.StatusCreated, gin.H{"conversation": conv, "message": pending})
	})

	// The user's conversations, most-recent first.
	g.GET("/conversations", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		page, _ := c.Get("paginate")
		items, total, err := models.ListConversations(ctx, user.ID, page.(database.Paginate))
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
	})

	// One conversation with its full message thread.
	g.GET("/conversations/:id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		conv, ok := ownedConversation(c, ctx, user.ID)
		if !ok {
			return
		}
		msgs, err := models.ListAIMessages(ctx, conv.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"conversation": conv, "messages": msgs})
	})

	// Send a user turn; the worker produces the assistant reply.
	g.POST("/conversations/:id/messages", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		conv, ok := ownedConversation(c, ctx, user.ID)
		if !ok {
			return
		}
		if !requireAI(c) {
			return
		}
		var form AIMessageForm
		if err := c.ShouldBindJSON(&form); err != nil {
			AbortValidation(c, err)
			return
		}
		pending, err := enqueueTurn(ctx, conv.ID, user.ID, form.Text)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusAccepted, pending)
	})

	// Report client-executed write results, then continue the conversation.
	g.POST("/conversations/:id/messages/:mid/result", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)
		conv, ok := ownedConversation(c, ctx, user.ID)
		if !ok {
			return
		}
		if !requireAI(c) {
			return
		}
		mid, err := uuid.Parse(c.Param("mid"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid message id")
			return
		}
		var form AIResultForm
		if err := c.ShouldBindJSON(&form); err != nil {
			AbortValidation(c, err)
			return
		}
		msg, err := models.GetAIMessage(ctx, mid, conv.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "message not found")
			return
		}

		var actions []storedAction
		_ = json.Unmarshal(msg.Actions, &actions)
		for i := range actions {
			if i >= len(form.Results) {
				break
			}
			r := form.Results[i]
			if r.OK {
				actions[i].Status = "done"
				actions[i].Result = r.Result
			} else {
				actions[i].Status = "failed"
				actions[i].Result = json.RawMessage(`{"error":` + jsonString(r.Error) + `}`)
			}
		}
		updated, _ := json.Marshal(actions)
		if err := models.SetMessageActions(ctx, mid, updated, models.AIStatusDone); err != nil {
			AbortServer(c, err)
			return
		}

		pending, err := continueTurn(ctx, conv.ID, user.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusAccepted, pending)
	})
}

// ownedConversation loads the :id conversation for the user, or aborts 404.
func ownedConversation(c *gin.Context, ctx context.Context, userID uuid.UUID) (models.AIConversation, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		AbortStatus(c, http.StatusBadRequest, "invalid conversation id")
		return models.AIConversation{}, false
	}
	conv, err := models.GetConversation(ctx, id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			AbortStatus(c, http.StatusNotFound, "conversation not found")
		} else {
			AbortServer(c, err)
		}
		return models.AIConversation{}, false
	}
	return conv, true
}

// enqueueTurn stores the user message + a pending assistant row and queues the
// worker to fill it. Returns the pending assistant message.
func enqueueTurn(ctx context.Context, convID, userID uuid.UUID, text string) (*models.AIMessage, error) {
	if _, err := models.InsertMessage(ctx, convID, models.AIRoleUser, text, models.AIStatusDone); err != nil {
		return nil, err
	}
	return continueTurn(ctx, convID, userID)
}

// continueTurn creates a pending assistant row and queues the worker (used after
// a user message and after client write-results come back).
func continueTurn(ctx context.Context, convID, userID uuid.UUID) (*models.AIMessage, error) {
	id, err := models.InsertMessage(ctx, convID, models.AIRoleAssistant, "", models.AIStatusPending)
	if err != nil {
		return nil, err
	}
	_ = models.TouchConversation(ctx, convID)
	events.EmitAI(convID, userID, id)
	return &models.AIMessage{ID: id, ConversationID: convID, Role: models.AIRoleAssistant, Status: models.AIStatusPending}, nil
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
