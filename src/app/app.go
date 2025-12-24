package app

import (
	"coachwise/src/app/views"
	"coachwise/src/config"
	"context"
	"fmt"
	"log"
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
			log.Printf("OpenAPI spec not found at %s: %v", absPath, err)
		}
	} else {
		log.Printf("could not resolve OpenAPI path %s: %v", openAPIPath, err)
	}

	// Ensure uploads directory exists and serve it statically
	uploadDir := filepath.Join(".", "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Fatalf("failed to create upload directory: %v", err)
	}
	router.Static("/uploads", uploadDir)

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
