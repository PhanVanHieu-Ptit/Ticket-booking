package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	appErrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const reserveLua = `
local session_id = KEYS[1]
local category = ARGV[1]
local ttl = tonumber(ARGV[2])

local purchased_set = "purchased:sessions"
local hold_key = "hold:" .. session_id
local available_set = "tickets:available:" .. category

-- 1. Verify session does not already exist in purchased:sessions
if redis.call("SISMEMBER", purchased_set, session_id) == 1 then
    return { "ERR", "PURCHASE_LIMIT_EXCEEDED" }
end

-- 2. Verify session does not already have an active hold
if redis.call("EXISTS", hold_key) == 1 then
    return { "ERR", "ACTIVE_HOLD_EXISTS" }
end

-- 3. Verify there is at least one ticket ID in the set
local count = redis.call("SCARD", available_set)
if count == 0 then
    return { "ERR", "TICKET_UNAVAILABLE" }
end

-- 5. Pop ticket ID and set hold key
local ticket_id = redis.call("SPOP", available_set)
if not ticket_id then
    return { "ERR", "TICKET_UNAVAILABLE" }
end

redis.call("SET", hold_key, ticket_id .. ":" .. category, "EX", ttl)
return { "OK", ticket_id }
`

type ReservationHandler struct {
	db        *sql.DB
	rdb       *redis.Client
	jwtSecret []byte
}

// NewReservationHandler creates a new instance of ReservationHandler.
func NewReservationHandler(db *sql.DB, rdb *redis.Client, jwtSecret []byte) *ReservationHandler {
	return &ReservationHandler{
		db:        db,
		rdb:       rdb,
		jwtSecret: jwtSecret,
	}
}

type reserveRequest struct {
	Category string `json:"category" binding:"required"`
}

// ReserveTicket handles POST /api/v1/tickets/reserve
func (h *ReservationHandler) ReserveTicket(c *gin.Context) {
	ctx := c.Request.Context()

	var req reserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(
			appErrors.ErrCodeInvalidInput,
			"Category is required",
			nil,
		))
		return
	}

	// 1. Validate requested category is VIP or Standard
	if req.Category != "VIP" && req.Category != "Standard" {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(
			appErrors.ErrCodeInvalidCategory,
			"The requested ticket category is invalid.",
			nil,
		))
		return
	}

	// 2. Fetch session ID from request context (set by SessionMiddleware)
	sessionIDVal, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(
			appErrors.ErrCodeSessionRequired,
			"Session token is required",
			nil,
		))
		return
	}
	sessionID := sessionIDVal.(string)

	// 3. Execute Redis Lua Script
	res, err := h.rdb.Eval(ctx, reserveLua, []string{sessionID}, req.Category, 300).Slice()
	if err != nil {
		logger.Error("Redis reservation script failed to execute", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeInternal,
			"Internal server error",
			nil,
		))
		return
	}

	if len(res) < 2 {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeInternal,
			"Invalid response from reservation shield",
			nil,
		))
		return
	}

	status, _ := res[0].(string)
	payload, _ := res[1].(string)

	if status == "ERR" {
		switch payload {
		case "PURCHASE_LIMIT_EXCEEDED":
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(
				appErrors.ErrCodePurchaseLimitExceed,
				"You have already purchased a ticket. Limit is 1 ticket per customer.",
				nil,
			))
		case "ACTIVE_HOLD_EXISTS":
			// Fetch current TTL from Redis
			ttlVal, _ := h.rdb.TTL(ctx, "hold:"+sessionID).Result()
			expiresAt := time.Now().Add(ttlVal)
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(
				appErrors.ErrCodeActiveHoldExists,
				"You already have an active reservation. Please complete your purchase or wait for it to expire.",
				gin.H{
					"expires_at":        expiresAt.Format(time.RFC3339),
					"seconds_remaining": int64(ttlVal.Seconds()),
				},
			))
		case "INVALID_CATEGORY":
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(
				appErrors.ErrCodeInvalidCategory,
				"The requested ticket category is invalid.",
				nil,
			))
		case "TICKET_UNAVAILABLE":
			c.JSON(http.StatusConflict, types.NewErrorResponse(
				appErrors.ErrCodeTicketUnavailable,
				"Sorry, all tickets in this category are currently reserved or sold. Please check back soon.",
				nil,
			))
		default:
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
				appErrors.ErrCodeInternal,
				"Internal reservation failure: "+payload,
				nil,
			))
		}
		return
	}

	ticketID, err := strconv.ParseInt(payload, 10, 64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeInternal,
			"Failed to parse ticket ID",
			nil,
		))
		return
	}

	// 4. Synchronize hold to PostgreSQL
	heldAt := time.Now()
	expiresAt := heldAt.Add(5 * time.Minute)

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("Failed to start database transaction for ticket hold", "error", err)
		h.rollbackRedis(ctx, sessionID, req.Category, ticketID)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeReservationFailed,
			"Failed to start reservation transaction",
			nil,
		))
		return
	}
	defer tx.Rollback() // Safe to call: no-op if committed

	// Clean up any expired holds for this session to prevent unique constraint violation
	_, err = tx.ExecContext(ctx, `
		UPDATE tickets 
		SET status = 'Available', session_id = NULL, held_at = NULL, expires_at = NULL, updated_at = NOW() 
		WHERE session_id = $1 AND status = 'Holding' AND expires_at <= NOW()
	`, sessionID)
	if err != nil {
		logger.Error("Failed to clean up expired holds for session during reservation", "error", err, "session_id", sessionID)
		tx.Rollback()
		h.rollbackRedis(ctx, sessionID, req.Category, ticketID)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeReservationFailed,
			"Reservation failed during database cleanup",
			nil,
		))
		return
	}

	// Update ticket status to 'Holding' where status is 'Available'
	result, err := tx.ExecContext(ctx, `
		UPDATE tickets 
		SET status = 'Holding', session_id = $2, held_at = $3, expires_at = $4, updated_at = NOW() 
		WHERE id = $1 AND status = 'Available'
	`, ticketID, sessionID, heldAt, expiresAt)

	if err != nil {
		// Query database for existing hold / purchase details to return precise user-facing error response
		var existingStatus string
		var existingExpiresAt sql.NullTime
		errFind := tx.QueryRowContext(ctx, "SELECT status, expires_at FROM tickets WHERE session_id = $1", sessionID).Scan(&existingStatus, &existingExpiresAt)
		if errFind == nil {
			tx.Rollback()
			h.rollbackRedis(ctx, sessionID, req.Category, ticketID)

			if existingStatus == "Sold" {
				c.JSON(http.StatusBadRequest, types.NewErrorResponse(
					appErrors.ErrCodePurchaseLimitExceed,
					"You have already purchased a ticket. Limit is 1 ticket per customer.",
					nil,
				))
				return
			} else if existingStatus == "Holding" {
				var secRem int64
				if existingExpiresAt.Valid {
					secRem = int64(time.Until(existingExpiresAt.Time).Seconds())
					if secRem < 0 {
						secRem = 0
					}
				}
				c.JSON(http.StatusBadRequest, types.NewErrorResponse(
					appErrors.ErrCodeActiveHoldExists,
					"You already have an active reservation. Please complete your purchase or wait for it to expire.",
					gin.H{
						"expires_at":        existingExpiresAt.Time.Format(time.RFC3339),
						"seconds_remaining": secRem,
					},
				))
				return
			}
		}

		logger.Error("Database update failed for ticket hold", "error", err, "ticket_id", ticketID)
		tx.Rollback()
		h.rollbackRedis(ctx, sessionID, req.Category, ticketID)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeReservationFailed,
			"Reservation failed during database update",
			nil,
		))
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		logger.Error("No rows affected on database update for ticket hold", "ticket_id", ticketID)
		tx.Rollback()
		h.rollbackRedis(ctx, sessionID, req.Category, ticketID)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeReservationFailed,
			"Reservation failed; ticket already reserved or unavailable",
			nil,
		))
		return
	}

	// Commit PostgreSQL transaction
	if err := tx.Commit(); err != nil {
		logger.Error("Database commit failed for ticket hold", "error", err)
		h.rollbackRedis(ctx, sessionID, req.Category, ticketID)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeReservationFailed,
			"Reservation failed during database commit",
			nil,
		))
		return
	}

	// 5. Query ticket code and price to return in response
	var ticketCode string
	var price float64
	err = h.db.QueryRowContext(ctx, "SELECT ticket_code, price FROM tickets WHERE id = $1", ticketID).Scan(&ticketCode, &price)
	if err != nil {
		logger.Warn("Failed to retrieve ticket details for response", "error", err, "ticket_id", ticketID)
	}

	// 6. Broadcast updated count to SSE broker
	newCount, err := h.rdb.SCard(ctx, "tickets:available:"+req.Category).Result()
	if err == nil {
		sse.BroadcastInventoryUpdate(req.Category, newCount)
	}

	c.JSON(http.StatusCreated, types.NewSuccessResponse(gin.H{
		"ticket_id":         ticketID,
		"ticket_code":       ticketCode,
		"category":          req.Category,
		"price":             price,
		"status":            "Holding",
		"held_at":           heldAt.Format(time.RFC3339),
		"expires_at":        expiresAt.Format(time.RFC3339),
		"seconds_remaining": 300,
	}))
}

// GetActiveHold handles GET /api/v1/tickets/hold
func (h *ReservationHandler) GetActiveHold(c *gin.Context) {
	ctx := c.Request.Context()

	sessionIDVal, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(
			appErrors.ErrCodeSessionRequired,
			"Session token is required",
			nil,
		))
		return
	}
	sessionID := sessionIDVal.(string)

	var ticketID int64
	var ticketCode string
	var category string
	var price float64
	var status string
	var heldAt, expiresAt time.Time

	// Query active hold where status is 'Holding' and expires_at is in the future
	err := h.db.QueryRowContext(ctx, `
		SELECT id, ticket_code, category, price, status, held_at, expires_at 
		FROM tickets 
		WHERE session_id = $1 AND status = 'Holding' AND expires_at > NOW()
	`, sessionID).Scan(&ticketID, &ticketCode, &category, &price, &status, &heldAt, &expiresAt)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, types.NewErrorResponse(
				appErrors.ErrCodeNoActiveHold,
				"No active reservation was found for this session.",
				nil,
			))
			return
		}
		logger.Error("Database query failed while fetching active hold", "error", err, "session_id", sessionID)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeInternal,
			"Internal server error",
			nil,
		))
		return
	}

	secondsRemaining := int64(time.Until(expiresAt).Seconds())
	if secondsRemaining < 0 {
		secondsRemaining = 0
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"ticket_id":         ticketID,
		"ticket_code":       ticketCode,
		"category":          category,
		"price":             price,
		"status":            status,
		"held_at":           heldAt.Format(time.RFC3339),
		"expires_at":        expiresAt.Format(time.RFC3339),
		"seconds_remaining": secondsRemaining,
	}))
}

func (h *ReservationHandler) rollbackRedis(ctx context.Context, sessionID string, category string, ticketID int64) {
	pipe := h.rdb.Pipeline()
	pipe.Del(ctx, "hold:"+sessionID)
	pipe.SAdd(ctx, "tickets:available:"+category, ticketID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.Error("Failed to execute Redis rollback pipeline", "error", err, "session_id", sessionID, "ticket_id", ticketID)
	}
}
