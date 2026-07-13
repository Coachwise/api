package app

import (
	"coachwise/src/app/views"
	"coachwise/src/app/ws"
	"coachwise/src/config"
	"coachwise/src/logger"
	"coachwise/src/storage"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Init() *gin.Engine {
	router := gin.Default()
	router.RedirectTrailingSlash = false

	// Serve OpenAPI spec for Swagger UI/clients (configurable path, default: openapi.yaml in repo root)
	openAPIPath := config.Config.OpenAPIPath
	if strings.TrimSpace(openAPIPath) == "" {
		openAPIPath = "openapi.yaml"
	}
	if absPath, err := filepath.Abs(openAPIPath); err == nil {
		if _, err := os.Stat(absPath); err == nil {
			router.StaticFile("/openapi.yaml", absPath)
		} else {
			logger.Errorf("OpenAPI spec not found at %s: %v", absPath, err)
		}
	} else {
		logger.Errorf("could not resolve OpenAPI path %s: %v", openAPIPath, err)
	}

	// Serve the local storage backend. A remote backend (S3/CDN) serves its own
	// files, so there is nothing to mount.
	if dir := storage.LocalDir(); dir != "" {
		router.Use(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/uploads/") {
				c.Header("X-Content-Type-Options", "nosniff")
			}
			c.Next()
		})
		router.Static("/uploads", dir)
	}

	// Realtime refetch-signal socket. Registered BEFORE the CORS/2s-timeout
	// middleware below so this long-lived connection isn't killed after 2s.
	router.GET("/ws", ws.Handle)

	// Configure CORS - allow configured origins or fall back to allow-all to avoid empty config panics
	corsConfig := cors.DefaultConfig()
	if len(config.Config.CORS.AllowedOrigins) > 0 {
		corsConfig.AllowOrigins = config.Config.CORS.AllowedOrigins
		corsConfig.AllowAllOrigins = false
	} else {
		corsConfig.AllowAllOrigins = true
	}

	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	corsConfig.ExposeHeaders = []string{"Content-Length"}
	corsConfig.AllowCredentials = true
	corsConfig.MaxAge = 12 * time.Hour

	// Apply CORS middleware
	router.Use(cors.New(corsConfig))

	router.Use(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		c.Set("ctx", ctx)
		c.Next()
	})

	views.Init(router)
	return router
}

func Serve() {
	router := Init()
	router.Run(fmt.Sprintf("127.0.0.1:%d", config.Config.Port))
}
