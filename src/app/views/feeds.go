package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"context"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func feedsGroup(router *gin.Engine) {
	g := router.Group("feeds")
	g.Use(auth.LoginRequired())

	g.POST("", func(c *gin.Context) {
		form := new(FeedCreateForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}

		if (form.Body == nil || strings.TrimSpace(*form.Body) == "") && len(form.Media) == 0 {
			AbortStatus(c, http.StatusBadRequest, "either body or media is required")
			return
		}

		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		visibility := "PUBLIC"
		if form.Visibility != nil && *form.Visibility != "" {
			visibility = strings.ToUpper(*form.Visibility)
		}

		feed := &models.Feed{
			UserID:     user.ID,
			Visibility: visibility,
		}

		if form.Body != nil {
			body := strings.TrimSpace(*form.Body)
			if body != "" {
				safeBody := html.EscapeString(body)
				feed.Body = &safeBody
			}
		}

		if form.Location != nil {
			loc := strings.TrimSpace(*form.Location)
			if loc != "" {
				safeLoc := html.EscapeString(loc)
				feed.Location = &safeLoc
			}
		}

		media := make([]models.FeedMedia, 0, len(form.Media))
		for _, m := range form.Media {
			media = append(media, models.FeedMedia{
				Kind:         strings.ToUpper(m.Kind),
				URL:          strings.TrimSpace(m.URL),
				ThumbnailURL: m.ThumbnailURL,
				OrderIndex:   m.OrderIndex,
			})
		}

		if err := feed.Create(ctx.(context.Context), media, form.Tags); err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, feed)
	})

	g.GET("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		feeds, total, err := models.ListFeeds(ctx.(context.Context), user.ID, limit, offset)
		if err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"feeds":       feeds,
			"total_count": total,
		})
	})

	g.GET("/:id", func(c *gin.Context) {
		feedID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid feed id")
			return
		}
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		feed, err := models.GetFeedWithDetails(ctx.(context.Context), feedID, user.ID)
		if err != nil {
			AbortStatus(c, http.StatusNotFound, "feed not found")
			return
		}

		c.JSON(http.StatusOK, feed)
	})

	g.DELETE("/:id", func(c *gin.Context) {
		feedID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid feed id")
			return
		}
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		if err := models.DeleteFeed(ctx.(context.Context), feedID, user.ID); err != nil {
			Abort(c, CodeNotFound)
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "feed deleted"})
	})

	g.POST("/:id/like", func(c *gin.Context) {
		feedID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid feed id")
			return
		}
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		if _, err := models.GetFeed(feedID); err != nil {
			AbortStatus(c, http.StatusNotFound, "feed not found")
			return
		}

		if err := models.AddFeedLike(ctx.(context.Context), feedID, user.ID); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "liked"})
	})

	g.DELETE("/:id/like", func(c *gin.Context) {
		feedID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid feed id")
			return
		}
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		if _, err := models.GetFeed(feedID); err != nil {
			AbortStatus(c, http.StatusNotFound, "feed not found")
			return
		}

		if err := models.RemoveFeedLike(ctx.(context.Context), feedID, user.ID); err != nil {
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "unliked"})
	})

	g.GET("/:id/comments", func(c *gin.Context) {
		feedID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid feed id")
			return
		}
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		if _, err := models.GetFeedWithDetails(ctx.(context.Context), feedID, user.ID); err != nil {
			AbortStatus(c, http.StatusNotFound, "feed not found")
			return
		}

		comments, err := models.ListFeedComments(ctx.(context.Context), feedID)
		if err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"comments": comments})
	})

	g.POST("/:id/comments", func(c *gin.Context) {
		feedID, err := uuid.Parse(c.Param("id"))
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "invalid feed id")
			return
		}
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		if _, err := models.GetFeedWithDetails(ctx.(context.Context), feedID, user.ID); err != nil {
			AbortStatus(c, http.StatusNotFound, "feed not found")
			return
		}

		form := new(FeedCommentForm)
		if err := c.ShouldBindJSON(form); err != nil {
			AbortValidation(c, err)
			return
		}
		body := strings.TrimSpace(form.Body)
		if body == "" {
			AbortStatus(c, http.StatusBadRequest, "comment body is required")
			return
		}

		comment := &models.FeedComment{
			FeedID: feedID,
			UserID: user.ID,
			Body:   html.EscapeString(body),
		}

		if err := comment.Create(ctx.(context.Context)); err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, comment)
	})
}
