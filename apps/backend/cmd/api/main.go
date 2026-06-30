package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env if present
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Set Gin mode
	ginMode := os.Getenv("GIN_MODE")
	if ginMode != "" {
		gin.SetMode(ginMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize Gin engine
	r := gin.New()

	// Use standard recovery and logging middleware
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

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
		// Health check endpoint
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":      "healthy",
				"version":     "1.0.0-foundation",
				"environment": os.Getenv("APP_ENV"),
				"services": gin.H{
					"database": "mocked_healthy",
					"redis":    "mocked_healthy",
				},
			})
		})
	}

	// Determine port
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Ticket Booking API Server on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
