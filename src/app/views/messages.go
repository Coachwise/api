package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func messageGroup(router *gin.Engine) {
	g := router.Group("messages")
	g.Use(auth.LoginRequired())

	// Send a message (DIRECT via recipient_id or CHANNEL via chat_id)
	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var form MessageForm
		if err := c.ShouldBindJSON(&form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var chatID uuid.UUID
		if form.RecipientID == nil && form.ChatID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "recipient_id or chat_id required"})
			return
		}

		ctx := c.MustGet("ctx")
		if form.RecipientID != nil {
			recipientID, err := uuid.Parse(strings.TrimSpace(*form.RecipientID))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recipient_id"})
				return
			}
			chat, err := models.GetOrCreateDirectChat(ctx.(context.Context), user.ID, recipientID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			chatID = chat.ID
		} else {
			id, err := uuid.Parse(strings.TrimSpace(*form.ChatID))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat_id"})
				return
			}
			chatID = id
		}

		if form.MediaID != nil {
			if _, err := uuid.Parse(*form.MediaID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media_id"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !canSend {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		msg, err := models.CreateMessage(ctx.(context.Context), chatID, user.ID, form.Body, mediaUUID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, msg)
	})

	// List messages with a peer (direct chat inferred)
	g.GET("/:peer_id", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		peerParam := strings.TrimSpace(c.Param("peer_id"))
		if peerParam == "threads" {
			limit, offset := parsePagination(c, 50, 200)
			threads, err := models.ListThreads(ctx.(context.Context), limit, offset)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, threads)
			return
		}

		peerID, err := uuid.Parse(peerParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid peer_id"})
			return
		}

		chat, err := models.GetOrCreateDirectChat(ctx.(context.Context), user.ID, peerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		limit, offset := parsePagination(c, 50, 200)
		msgs, err := models.ListMessages(ctx.(context.Context), chat.ID, limit, offset)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid peer_id"})
			return
		}

		chat, err := models.GetOrCreateDirectChat(ctx.(context.Context), user.ID, peerID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := models.MarkChatRead(ctx.(context.Context), chat.ID, user.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})
}
