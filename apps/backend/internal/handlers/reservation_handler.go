package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	appredis "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/redis"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	appErrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

const ticketHoldTTL = 5 * time.Minute

type ReservationHandler struct {
	db        *sql.DB
	redisSvc  *appredis.RedisService
	jwtSecret []byte
}

// NewReservationHandler creates a new instance of ReservationHandler.
func NewReservationHandler(db *sql.DB, redisSvc *appredis.RedisService, jwtSecret []byte) *ReservationHandler {
	return &ReservationHandler{
		db:        db,
		redisSvc:  redisSvc,
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

	// 3. Acquire an atomic Redis-backed hold BEFORE touching the database.
	// RedisService.HoldTicket runs a single Lua script that checks the
	// purchase limit, checks for an existing hold, and pops one ticket ID
	// from the category's available set, all atomically. If Redis reports
	// the session is ineligible or the category is sold out, we fail
	// immediately without any DB write.
	ticketID, err := h.redisSvc.HoldTicket(ctx, sessionID, req.Category, ticketHoldTTL)
	if err != nil {
		switch {
		case errors.Is(err, appredis.ErrPurchaseLimitExceeded):
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(
				appErrors.ErrCodePurchaseLimitExceed,
				"You have already purchased a ticket. Limit is 1 ticket per customer.",
				nil,
			))
		case errors.Is(err, appredis.ErrActiveHoldExists):
			ttlVal, _ := h.redisSvc.GetHoldTTL(ctx, sessionID)
			expiresAt := time.Now().Add(ttlVal)
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(
				appErrors.ErrCodeActiveHoldExists,
				"You already have an active reservation. Please complete your purchase or wait for it to expire.",
				gin.H{
					"expires_at":        expiresAt.Format(time.RFC3339),
					"seconds_remaining": int64(ttlVal.Seconds()),
				},
			))
		case errors.Is(err, appredis.ErrTicketUnavailable):
			c.JSON(http.StatusConflict, types.NewErrorResponse(
				appErrors.ErrCodeTicketUnavailable,
				"Sorry, all tickets in this category are currently reserved or sold. Please check back soon.",
				nil,
			))
		default:
			logger.Error("Redis reservation hold failed to execute", "error", err, "session_id", sessionID)
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
				appErrors.ErrCodeInternal,
				"Internal server error",
				nil,
			))
		}
		return
	}

	// 4. Synchronize hold to PostgreSQL — second layer of protection.
	heldAt := time.Now()
	expiresAt := heldAt.Add(ticketHoldTTL)

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("Failed to start database transaction for ticket hold", "error", err)
		h.redisSvc.ReleaseHold(ctx, sessionID, req.Category, ticketID)
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
		h.redisSvc.ReleaseHold(ctx, sessionID, req.Category, ticketID)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeReservationFailed,
			"Reservation failed during database cleanup",
			nil,
		))
		return
	}

	// Lock the ticket row (SELECT ... FOR UPDATE) before deciding whether to
	// write it. This holds a row lock for the rest of the transaction, so no
	// other transaction can concurrently read/modify this specific ticket
	// until we commit or roll back.
	var currentStatus string
	var currentExpiresAt sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT status, expires_at
		FROM tickets
		WHERE id = $1
		FOR UPDATE
	`, ticketID).Scan(&currentStatus, &currentExpiresAt)
	if err != nil {
		logger.Error("Failed to lock ticket row for hold", "error", err, "ticket_id", ticketID)
		tx.Rollback()
		h.redisSvc.ReleaseHold(ctx, sessionID, req.Category, ticketID)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			appErrors.ErrCodeReservationFailed,
			"Reservation failed while locking ticket row",
			nil,
		))
		return
	}

	if currentStatus != "Available" {
		tx.Rollback()
		h.redisSvc.ReleaseHold(ctx, sessionID, req.Category, ticketID)

		if currentStatus == "Sold" {
			c.JSON(http.StatusBadRequest, types.NewErrorResponse(
				appErrors.ErrCodePurchaseLimitExceed,
				"You have already purchased a ticket. Limit is 1 ticket per customer.",
				nil,
			))
			return
		}

		// currentStatus == "Holding"
		var secRem int64
		if currentExpiresAt.Valid {
			secRem = int64(time.Until(currentExpiresAt.Time).Seconds())
			if secRem < 0 {
				secRem = 0
			}
		}
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(
			appErrors.ErrCodeActiveHoldExists,
			"You already have an active reservation. Please complete your purchase or wait for it to expire.",
			gin.H{
				"expires_at":        currentExpiresAt.Time.Format(time.RFC3339),
				"seconds_remaining": secRem,
			},
		))
		return
	}

	// Update ticket status to 'Holding'. The row is already locked and
	// confirmed 'Available' above; the WHERE clause is kept as a defensive,
	// belt-and-suspenders guard against unexpected state changes.
	result, err := tx.ExecContext(ctx, `
		UPDATE tickets
		SET status = 'Holding', session_id = $2, held_at = $3, expires_at = $4, updated_at = NOW()
		WHERE id = $1 AND status = 'Available'
	`, ticketID, sessionID, heldAt, expiresAt)
	if err != nil {
		logger.Error("Database update failed for ticket hold", "error", err, "ticket_id", ticketID)
		tx.Rollback()
		h.redisSvc.ReleaseHold(ctx, sessionID, req.Category, ticketID)
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
		h.redisSvc.ReleaseHold(ctx, sessionID, req.Category, ticketID)
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
		h.redisSvc.ReleaseHold(ctx, sessionID, req.Category, ticketID)
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
	newCount, err := h.redisSvc.AvailableCount(ctx, req.Category)
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
		"seconds_remaining": int64(ticketHoldTTL.Seconds()),
		"server_time":       time.Now().Format(time.RFC3339),
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

	now := time.Now()
	secondsRemaining := int64(expiresAt.Sub(now).Seconds())
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
		"server_time":       now.Format(time.RFC3339),
	}))
}
