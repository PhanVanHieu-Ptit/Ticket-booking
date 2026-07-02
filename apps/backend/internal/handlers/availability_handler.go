package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	appErrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// AvailabilityHandler manages routes related to ticket count queries and SSE live streams.
type AvailabilityHandler struct {
	db        *sql.DB
	rdb       *redis.Client
	broker    *sse.Broker
	jwtSecret []byte
}

// NewAvailabilityHandler instantiates an AvailabilityHandler.
func NewAvailabilityHandler(db *sql.DB, rdb *redis.Client, broker *sse.Broker, jwtSecret []byte) *AvailabilityHandler {
	return &AvailabilityHandler{
		db:        db,
		rdb:       rdb,
		broker:    broker,
		jwtSecret: jwtSecret,
	}
}

// GetAvailability retrieves the counts of remaining tickets directly from Redis available sets.
func (h *AvailabilityHandler) GetAvailability(c *gin.Context) {
	ctx := c.Request.Context()

	// Query VIP and Standard available counts via SCARD
	vipCount, err := h.rdb.SCard(ctx, "tickets:available:VIP").Result()
	if err != nil {
		logger.Error("Failed to fetch available VIP count from Redis", "error", err)
		c.Error(appErrors.NewInternal(err, "Internal server error"))
		return
	}

	stdCount, err := h.rdb.SCard(ctx, "tickets:available:Standard").Result()
	if err != nil {
		logger.Error("Failed to fetch available Standard count from Redis", "error", err)
		c.Error(appErrors.NewInternal(err, "Internal server error"))
		return
	}

	vipStatus := "Available"
	if vipCount <= 0 {
		vipStatus = "Sold Out"
	}

	stdStatus := "Available"
	if stdCount <= 0 {
		stdStatus = "Sold Out"
	}

	response := gin.H{
		"event_name":     "Neon Symphony: Hyperion Tour 2026",
		"total_capacity": 500,
		"categories": []gin.H{
			{
				"name":      "VIP",
				"price":     100.0,
				"available": vipCount,
				"total":     100,
				"status":    vipStatus,
			},
			{
				"name":      "Standard",
				"price":     50.0,
				"available": stdCount,
				"total":     400,
				"status":    stdStatus,
			},
		},
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(response))
}

// StreamAvailability establishes a persistent SSE stream to push real-time availability updates.
func (h *AvailabilityHandler) StreamAvailability(c *gin.Context) {
	// Extract session token from cookie or query parameter
	var tokenStr string
	if cookieVal, err := c.Cookie("session_token"); err == nil && cookieVal != "" {
		tokenStr = cookieVal
	} else if qVal := c.Query("session_token"); qVal != "" {
		tokenStr = qVal
	}

	// Session token validation is mandatory
	if tokenStr == "" {
		c.Error(appErrors.New(http.StatusBadRequest, appErrors.ErrCodeInvalidSessionToken, "Session token is required to establish stream"))
		c.Abort()
		return
	}

	_, _, err := session.VerifySessionToken(tokenStr, h.jwtSecret)
	if err != nil {
		c.Error(appErrors.New(http.StatusBadRequest, appErrors.ErrCodeInvalidSessionToken, "Invalid or expired session token"))
		c.Abort()
		return
	}

	// Set required headers for Server-Sent Events (SSE)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	// Subscribe client connection channel with the broker
	clientChan := h.broker.AddClient()
	defer h.broker.RemoveClient(clientChan)

	// Fetch and broadcast initial state immediately after establishing connection
	ctx := c.Request.Context()
	vipCount, err := h.rdb.SCard(ctx, "tickets:available:VIP").Result()
	if err != nil {
		logger.Error("Failed to fetch VIP available counts for initial state", "error", err)
		return
	}
	stdCount, err := h.rdb.SCard(ctx, "tickets:available:Standard").Result()
	if err != nil {
		logger.Error("Failed to fetch Standard available counts for initial state", "error", err)
		return
	}

	vipStatus := "Available"
	if vipCount <= 0 {
		vipStatus = "Sold Out"
	}
	stdStatus := "Available"
	if stdCount <= 0 {
		stdStatus = "Sold Out"
	}

	initialState := map[string]interface{}{
		"VIP": map[string]interface{}{
			"available": vipCount,
			"status":    vipStatus,
		},
		"Standard": map[string]interface{}{
			"available": stdCount,
			"status":    stdStatus,
		},
	}
	initialData, _ := json.Marshal(initialState)

	fmt.Fprintf(c.Writer, "event: initial_state\ndata: %s\n\n", initialData)
	c.Writer.Flush()

	// 15-second heartbeat ticker to prevent socket closure from idle middleware or proxies
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Stream execution loop
	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-clientChan:
			if !ok {
				return false
			}
			fmt.Fprintf(w, "%s", msg)
			return true
		case <-ticker.C:
			fmt.Fprintf(w, ": keep-alive\n\n")
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
