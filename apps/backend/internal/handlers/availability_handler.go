package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/constants"
	appErrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
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

// categoryDefinition represents a ticket category's static attributes as
// derived from the tickets table (name, price, total inventory).
type categoryDefinition struct {
	Name  string
	Price float64
	Total int
}

// listCategoryDefinitions returns the distinct ticket categories currently
// present in the tickets table, along with their price and total inventory.
// This is the single source of truth for "which categories exist" — nothing
// about the category list is hardcoded in Go.
func (h *AvailabilityHandler) listCategoryDefinitions(ctx context.Context) ([]categoryDefinition, error) {
	rows, err := h.db.QueryContext(ctx, `
		SELECT category, MIN(price) AS price, COUNT(*) AS total
		FROM tickets
		GROUP BY category
		ORDER BY MIN(price) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []categoryDefinition
	for rows.Next() {
		var d categoryDefinition
		if err := rows.Scan(&d.Name, &d.Price, &d.Total); err != nil {
			return nil, err
		}
		defs = append(defs, d)
	}
	return defs, rows.Err()
}

// GetAvailability retrieves the list of ticket categories from the database
// and their remaining counts from Redis available sets.
func (h *AvailabilityHandler) GetAvailability(c *gin.Context) {
	ctx := c.Request.Context()

	defs, err := h.listCategoryDefinitions(ctx)
	if err != nil {
		logger.Error("Failed to fetch ticket categories from database", "error", err)
		c.Error(appErrors.NewInternal(err, "Internal server error"))
		return
	}

	categories := make([]gin.H, 0, len(defs))
	totalCapacity := 0
	for _, d := range defs {
		available, err := h.rdb.SCard(ctx, constants.RedisKeyAvailableTickets(d.Name)).Result()
		if err != nil {
			logger.Error("Failed to fetch available count from Redis", "error", err, "category", d.Name)
			c.Error(appErrors.NewInternal(err, "Internal server error"))
			return
		}

		status := "Available"
		if available <= 0 {
			status = "Sold Out"
		}

		categories = append(categories, gin.H{
			"name":      d.Name,
			"price":     d.Price,
			"available": available,
			"total":     d.Total,
			"status":    status,
		})
		totalCapacity += d.Total
	}

	response := gin.H{
		"event_name":     "Neon Symphony: Hyperion Tour 2026",
		"total_capacity": totalCapacity,
		"categories":     categories,
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
	defs, err := h.listCategoryDefinitions(ctx)
	if err != nil {
		logger.Error("Failed to fetch ticket categories for initial state", "error", err)
		return
	}

	initialState := make(map[string]interface{}, len(defs))
	for _, d := range defs {
		available, err := h.rdb.SCard(ctx, constants.RedisKeyAvailableTickets(d.Name)).Result()
		if err != nil {
			logger.Error("Failed to fetch available counts for initial state", "error", err, "category", d.Name)
			return
		}

		status := "Available"
		if available <= 0 {
			status = "Sold Out"
		}

		initialState[d.Name] = map[string]interface{}{
			"available": available,
			"status":    status,
		}
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
