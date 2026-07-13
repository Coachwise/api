package views

import (
	"coachwise/src/app/auth"
	"coachwise/src/app/models"
	"coachwise/src/config"
	"coachwise/src/storage"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type mediaForm struct {
	URL       string `json:"url" binding:"required"`
	Filename  string `json:"filename" binding:"required"`
	SizeBytes int64  `json:"size_bytes"`
}

// sniffMediaType reads the first bytes to decide what the file really is, then
// rewinds so the caller can still store it whole.
func sniffMediaType(file multipart.File) (string, error) {
	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("file could not be read")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("file could not be read")
	}

	contentType := http.DetectContentType(head[:n])
	if err := storage.CheckMediaType(contentType); err != nil {
		return "", err
	}
	return contentType, nil
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

	// Multipart upload: hands the file to the storage service and records it.
	g.POST("/upload", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)
		ctx := c.MustGet("ctx").(context.Context)

		header, err := c.FormFile("file")
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "file is required")
			return
		}
		if header.Size > storage.MaxSize() {
			AbortStatus(c, http.StatusRequestEntityTooLarge, storage.ErrTooLarge.Error())
			return
		}

		file, err := header.Open()
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, "file could not be read")
			return
		}
		defer file.Close()

		// Sniff the type from the bytes; the client's Content-Type is a claim,
		// not evidence.
		contentType, err := sniffMediaType(file)
		if err != nil {
			AbortStatus(c, http.StatusBadRequest, err.Error())
			return
		}

		key := storage.Key(storage.KindMedia, header.Filename)
		if err := storage.Get().Put(ctx, key, file, header.Size, contentType); err != nil {
			AbortServer(c, err)
			return
		}

		url := storage.Get().URL(key)
		media, err := models.CreateMedia(ctx, user.ID, url, filepath.Base(header.Filename), header.Size)
		if err != nil {
			AbortServer(c, err)
			return
		}

		c.JSON(http.StatusOK, media)
	})

	g.POST("", func(c *gin.Context) {
		user := c.MustGet("user").(*models.User)

		var form mediaForm
		if err := c.ShouldBindJSON(&form); err != nil {
			AbortValidation(c, err)
			return
		}
		form.URL = resolveMediaURL(c, form.URL)

		ctx := c.MustGet("ctx")
		media, err := models.CreateMedia(ctx.(context.Context), user.ID, form.URL, form.Filename, form.SizeBytes)
		if err != nil {
			AbortServer(c, err)
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
			AbortServer(c, err)
			return
		}
		c.JSON(http.StatusOK, items)
	})
}
