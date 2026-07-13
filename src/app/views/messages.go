package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/events"
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"coachwise/src/database"
)

// requireConnection guards direct-message operations: only connected users may
// message each other. Writes a 403 and returns false when not connected.
func requireConnection(c *gin.Context, ctx context.Context, me, peer uuid.UUID) bool {
	connected, err := models.IsConnected(ctx, me, peer)
	if err != nil {
		AbortServer(c, err)
		return false
	}
	if !connected {
		Abort(c, CodeNotConnected)
		return false
	}
	return true
}

func messageGroup(router *gin.Engine) {
	g := router.Group("messages")
	g.Use(auth.LoginRequired())

	// Send a message (DIRECT via recipient_id or CHANNEL via chat_id)
	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var form MessageForm
		if err := c.ShouldBindJSON(&form); err != nil {
			AbortValidation(c, err)
			return
		}

		var chatID uuid.UUID
		var directPeer *uuid.UUID // set for direct messages, to poke the recipient live
		if form.RecipientID == nil && form.ChatID == nil {
			AbortStatus(c, http.StatusBadRequest, "recipient_id or chat_id required")
			return
		}

		ctx := c.MustGet("ctx")
		if form.RecipientID != nil {
			recipientID, err := uuid.Parse(strings.TrimSpace(*form.RecipientID))
			if err != nil {
				AbortStatus(c, http.StatusBadRequest, "invalid recipient_id")
				return
			}
			if !requireConnection(c, ctx.(context.Context), user.ID, recipientID) {
				return
			}
			chat, err := models.GetOrCreateDirectChat(ctx.(context.Context), user.ID, recipientID)
			if err != nil {
				AbortServer(c, err)
				return
			}
			chatID = chat.ID
			directPeer = &recipientID
		} else {
			id, err := uuid.Parse(strings.TrimSpace(*form.ChatID))
			if err != nil {
				AbortStatus(c, http.StatusBadRequest, "invalid chat_id")
				return
			}
			chatID = id
		}

		if form.MediaID != nil {
			if _, err := uuid.Parse(*form.MediaID); err != nil {
				AbortStatus(c, http.StatusBadRequest, "invalid media_id")
				return
			}
		}

		var mediaUUID *uuid.UUID
		if form.MediaID != nil {
			id := uuid.MustParse(*form.MediaID)
			mediaUUID = &id
		}

		canSend, err := models.CanSend(ctx.(context.Context), chatID, user.ID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		if !canSend {
			AbortStatus(c, http.StatusForbidden, "forbidden")
			return
		}

		msg, err := models.CreateMessage(ctx.(context.Context), chatID, user.ID, form.Body, mediaUUID)
		if err != nil {
			AbortServer(c, err)
			return
		}
		if directPeer != nil {
			events.EmitSignal(*directPeer, "messages") // recipient's thread refreshes live
		}
		c.JSON(http.StatusOK, msg)
	})

	// List messages with a peer (direct chat inferred)
	g.GET("/:peer_id", paginate(), func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		peerParam := strings.TrimSpace(c.Param("peer_id"))
		if peerParam == "threads" {
			page, _ := c.Get("paginate")
			threads, total, err := models.ListThreads(ctx.(context.Context), user.ID, page.(database.Paginate))
			if err != nil {
				AbortServer(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": threads, "total": total})
			return
		}

		peerID, err := uuid.Parse(peerParam)
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid peer_id")
			return
		}
		if !requireConnection(c, ctx.(context.Context), user.ID, peerID) {
			return
		}

		chat, err := models.GetOrCreateDirectChat(ctx.(context.Context), user.ID, peerID)
		if err != nil {
			AbortServer(c, err)
			return
		}

		limit, offset := parsePagination(c, 50, 200)
		msgs, err := models.ListMessages(ctx.(context.Context), chat.ID, limit, offset)
		if err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, msgs)
	})

	// Mark chat read for direct peer
	g.POST("/:peer_id/read", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		peerParam := strings.TrimSpace(c.Param("peer_id"))
		if peerParam == "threads" {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
			return
		}

		peerID, err := uuid.Parse(peerParam)
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid peer_id")
			return
		}
		if !requireConnection(c, ctx.(context.Context), user.ID, peerID) {
			return
		}

		chat, err := models.GetOrCreateDirectChat(ctx.(context.Context), user.ID, peerID)
		if err != nil {
			AbortServer(c, err)
			return
		}

		if err := models.MarkChatRead(ctx.(context.Context), chat.ID, user.ID); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
}
