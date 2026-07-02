package main

import (
	"context"
	"log"
	"net/http"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/handlers"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/middleware"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/modules/payment"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/modules/reclamation"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/redis"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/config"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/db"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
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

	// Initialize Redis connection pool
	rdb, err := redis.Init(cfg.RedisURL)
	if err != nil {
		logger.Error("Failed to initialize Redis connection pool", "error", err)
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	logger.Info("Redis connection pool initialized successfully")

	// Synchronize ticket inventory from Postgres to Redis on startup
	if err := redis.SyncInventory(database); err != nil {
		logger.Error("Failed to synchronize ticket inventory to Redis", "error", err)
		log.Fatalf("Failed to synchronize ticket inventory: %v", err)
	}

	// Initialize and start SSE event broker
	sse.GlobalBroker = sse.NewBroker()
	sse.GlobalBroker.Start()
	logger.Info("SSE Event Broker started successfully")

	// Initialize and start Reclamation module background workers
	reclaimModule := reclamation.NewModule(database, rdb)
	reclaimModule.StartBackgroundJobs(context.Background())
	logger.Info("Reclamation background workers started successfully")

	// Set Gin mode based on config
	gin.SetMode(cfg.GinMode)

	// Global request validation: reject any JSON body field that isn't
	// declared on the target DTO struct, instead of silently ignoring it
	// (equivalent to a ValidationPipe with whitelist/forbidNonWhitelisted).
	binding.EnableDecoderDisallowUnknownFields = true

	// Initialize Gin engine
	r := gin.New()

	// Use custom recovery and logging middleware
	r.Use(logger.RecoveryMiddleware(cfg.IsProduction()))
	r.Use(logger.GinMiddleware())

	// Global exception filter: every handler/middleware reports errors via
	// c.Error(err) instead of writing its own JSON; this is the only place
	// that serializes an error response for non-panic failures.
	r.Use(middleware.ErrorHandlerMiddleware(cfg.IsProduction()))

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

	// Initialize the Redis-backed ticket inventory service
	redisSvc := redis.NewRedisService(rdb)

	// Initialize handlers
	sessionHandler := handlers.NewSessionHandler()
	adminHandler := handlers.NewAdminHandler(database, []byte(cfg.JWTSecret))
	availabilityHandler := handlers.NewAvailabilityHandler(database, rdb, sse.GlobalBroker, []byte(cfg.JWTSecret))
	reservationHandler := handlers.NewReservationHandler(database, redisSvc, []byte(cfg.JWTSecret))
	paymentModule := payment.NewModule(database, rdb)

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

			redisStatus := "healthy"
			if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
				redisStatus = "unhealthy"
				logger.Error("Redis health check ping failed", "error", err)
			}

			c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
				"status":      "healthy",
				"version":     "1.0.0-foundation",
				"environment": cfg.AppEnv,
				"services": gin.H{
					"database": dbStatus,
					"redis":    redisStatus,
				},
			}))
		})
	}

	// v1 API route group with session middleware
	v1 := r.Group("/api/v1")
	v1.Use(middleware.SessionMiddleware(cfg.JWTSecret, cfg.IsProduction()))
	{
		// Sessions endpoint
		v1.POST("/sessions", sessionHandler.InitializeSession)

		// Availability endpoints
		v1.GET("/tickets/availability", availabilityHandler.GetAvailability)
		v1.GET("/tickets/availability/stream", availabilityHandler.StreamAvailability)

		// Reservation endpoints
		v1.POST("/tickets/reserve", reservationHandler.ReserveTicket)
		v1.GET("/tickets/hold", reservationHandler.GetActiveHold)
		v1.POST("/tickets/hold/cancel", reservationHandler.CancelHold)

		// Payment endpoints
		v1.POST("/payments/checkout", middleware.IdempotencyMiddleware(rdb), paymentModule.Controller.Checkout)

		// Test-only endpoints: let e2e/integration tests deterministically
		// manipulate inventory (e.g. force a category down to a single seat
		// to exercise oversell/race-condition scenarios). Never mounted in
		// production, since these mutate/destroy ticket inventory state.
		if !cfg.IsProduction() {
			testHandler := handlers.NewTestHandler(database)
			v1.POST("/test/reset-inventory", testHandler.ResetInventory)
			logger.Info("Test-only routes mounted", "route", "/api/v1/test/reset-inventory", "app_env", cfg.AppEnv)
		}

		// Admin route group
		admin := v1.Group("/admin")
		{
			// Login is unprotected by admin auth middleware (but has session middleware)
			admin.POST("/login", adminHandler.Login)

			// Protect all other admin routes
			adminAuth := admin.Group("")
			adminAuth.Use(middleware.AdminAuthMiddleware(cfg.JWTSecret))
			{
				adminAuth.GET("/metrics", adminHandler.GetMetrics)
				adminAuth.GET("/holds", adminHandler.GetHolds)
			}
		}
	}

	logger.Info("Server is running", "addr", ":"+cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		logger.Error("Failed to start server", "error", err)
		log.Fatalf("Failed to start server: %v", err)
	}
}
