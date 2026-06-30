package main

import (
	"log"
	"net/http"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/config"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/db"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize structured logger
	logger.Init(cfg.AppEnv)
	logger.Info("Starting Ticket Booking API Server...", "env", cfg.AppEnv, "port", cfg.Port)

	// Initialize database connection pool
	database, err := db.Init(cfg.DatabaseURL)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection pool", "error", err)
		}
	}()
	logger.Info("Database connection pool initialized successfully")

	// Set Gin mode based on config
	gin.SetMode(cfg.GinMode)

	// Initialize Gin engine
	r := gin.New()

	// Use custom recovery and logging middleware
	r.Use(logger.RecoveryMiddleware())
	r.Use(logger.GinMiddleware())

	// Custom CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Base API route group
	api := r.Group("/api")
	{
		// Health check endpoint using standard response envelope
		api.GET("/health", func(c *gin.Context) {
			dbStatus := "healthy"
			if err := database.Ping(); err != nil {
				dbStatus = "unhealthy"
				logger.Error("Database health check ping failed", "error", err)
			}

			c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
				"status":      "healthy",
				"version":     "1.0.0-foundation",
				"environment": cfg.AppEnv,
				"services": gin.H{
					"database": dbStatus,
					"redis":    "mocked_healthy",
				},
			}))
		})
	}

	logger.Info("Server is running", "addr", ":"+cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		logger.Error("Failed to start server", "error", err)
		log.Fatalf("Failed to start server: %v", err)
	}
}

