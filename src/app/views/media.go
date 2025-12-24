package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/config"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mediaForm struct {
	URL       string `json:"url" binding:"required"`
	Filename  string `json:"filename" binding:"required"`
	SizeBytes int64  `json:"size_bytes"`
}

func resolveMediaURL(c *gin.Context, url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}

	base := strings.TrimSuffix(config.Config.MediaBaseURL, "/")
	if base == "" {
		scheme := "http"
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		} else if c.Request.TLS != nil {
			scheme = "https"
		}
		if host := c.Request.Host; host != "" {
			base = fmt.Sprintf("%s://%s", scheme, host)
		}
	}

	if base == "" {
		return url
	}
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	return base + url
}

func mediaGroup(router *gin.Engine) {
	g := router.Group("media")
	g.Use(auth.LoginRequired())

	// Multipart upload: saves file locally to ./uploads and records media entry
	g.POST("/upload", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx")

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}

		uploadDir := filepath.Join(".", "uploads")
		storedName := uuid.NewString() + filepath.Ext(file.Filename)
		dst := filepath.Join(uploadDir, storedName)

		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}

		url := resolveMediaURL(c, "/uploads/"+storedName)
		media, err := models.CreateMedia(ctx.(context.Context), user.ID, url, storedName, file.Size)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, media)
	})

	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var form mediaForm
		if err := c.ShouldBindJSON(&form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		form.URL = resolveMediaURL(c, form.URL)

		ctx := c.MustGet("ctx")
		media, err := models.CreateMedia(ctx.(context.Context), user.ID, form.URL, form.Filename, form.SizeBytes)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, media)
	})

	g.GET("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		ctx := c.MustGet("ctx")

		limit, offset := parsePagination(c, 50, 200)
		items, err := models.ListMedia(ctx.(context.Context), user.ID, limit, offset)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, items)
	})
}
