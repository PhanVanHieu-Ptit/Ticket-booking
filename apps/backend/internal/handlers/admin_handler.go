package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	db        *sql.DB
	jwtSecret []byte
}

// NewAdminHandler creates a new instance of AdminHandler.
func NewAdminHandler(db *sql.DB, jwtSecret []byte) *AdminHandler {
	return &AdminHandler{
		db:        db,
		jwtSecret: jwtSecret,
	}
}

type loginRequest struct {
	Passcode string `json:"passcode" binding:"required"`
}

// Login validates the passcode and returns a signed admin JWT token.
func (h *AdminHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.NewErrorResponse(
			errors.ErrCodeInvalidInput,
			"Passcode is required",
			nil,
		))
		return
	}

	// Query database for admin passcode hash
	var hashedPasscode string
	err := h.db.QueryRow("SELECT value FROM admin_configs WHERE key = 'admin_passcode'").Scan(&hashedPasscode)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Error("Admin passcode configuration not found in database")
			c.JSON(http.StatusUnauthorized, types.NewErrorResponse(
				errors.ErrCodeAdminUnauthorized,
				"Invalid admin passcode.",
				nil,
			))
			return
		}
		logger.Error("Database query failed while fetching admin passcode", "error", err)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			errors.ErrCodeInternal,
			"An unexpected database error occurred",
			nil,
		))
		return
	}

	// Compare passcodes
	err = bcrypt.CompareHashAndPassword([]byte(hashedPasscode), []byte(req.Passcode))
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.NewErrorResponse(
			errors.ErrCodeAdminUnauthorized,
			"Invalid admin passcode.",
			nil,
		))
		return
	}

	// Sign admin JWT
	tokenStr, expiresAt, err := session.SignAdminToken(h.jwtSecret)
	if err != nil {
		logger.Error("Failed to sign admin JWT token", "error", err)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			errors.ErrCodeInternal,
			"An unexpected error occurred during login",
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"token":      tokenStr,
		"expires_at": expiresAt.Format(time.RFC3339),
	}))
}

type InventoryBreakdown struct {
	VIP      int `json:"VIP"`
	Standard int `json:"Standard"`
}

type MetricsResponse struct {
	TotalTicketsSold   int                `json:"total_tickets_sold"`
	TotalRevenue       float64            `json:"total_revenue"`
	RemainingInventory InventoryBreakdown `json:"remaining_inventory"`
	HeldInventory      InventoryBreakdown `json:"held_inventory"`
	AvailableInventory InventoryBreakdown `json:"available_inventory"`
}

type HoldDetail struct {
	TicketID         int     `json:"ticket_id"`
	TicketCode       string  `json:"ticket_code"`
	Category         string  `json:"category"`
	Price            float64 `json:"price"`
	SessionID        string  `json:"session_id"`
	ExpiresAt        string  `json:"expires_at"`
	SecondsRemaining int     `json:"seconds_remaining"`
}

// GetMetrics returns statistics on ticket sales, revenue, and inventories.
func (h *AdminHandler) GetMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	var totalTicketsSold int
	err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tickets WHERE status = 'Sold'").Scan(&totalTicketsSold)
	if err != nil {
		logger.Error("Failed to fetch total tickets sold", "error", err)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			errors.ErrCodeInternal,
			"An unexpected database error occurred",
			nil,
		))
		return
	}

	var totalRevenue float64
	err = h.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(price), 0) FROM tickets WHERE status = 'Sold'").Scan(&totalRevenue)
	if err != nil {
		logger.Error("Failed to fetch total revenue", "error", err)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			errors.ErrCodeInternal,
			"An unexpected database error occurred",
			nil,
		))
		return
	}

	rows, err := h.db.QueryContext(ctx, "SELECT category, status, COUNT(*) FROM tickets GROUP BY category, status")
	if err != nil {
		logger.Error("Failed to fetch inventory breakdown", "error", err)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			errors.ErrCodeInternal,
			"An unexpected database error occurred",
			nil,
		))
		return
	}
	defer rows.Close()

	var (
		availVIP, availStd int
		heldVIP, heldStd   int
	)

	for rows.Next() {
		var category, status string
		var count int
		if err := rows.Scan(&category, &status, &count); err != nil {
			logger.Error("Failed to scan inventory breakdown row", "error", err)
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
				errors.ErrCodeInternal,
				"An unexpected database error occurred",
				nil,
			))
			return
		}
		switch category {
		case "VIP":
			switch status {
			case "Available":
				availVIP = count
			case "Holding":
				heldVIP = count
			}
		case "Standard":
			switch status {
			case "Available":
				availStd = count
			case "Holding":
				heldStd = count
			}
		}
	}

	metrics := MetricsResponse{
		TotalTicketsSold: totalTicketsSold,
		TotalRevenue:     totalRevenue,
		RemainingInventory: InventoryBreakdown{
			VIP:      availVIP + heldVIP,
			Standard: availStd + heldStd,
		},
		HeldInventory: InventoryBreakdown{
			VIP:      heldVIP,
			Standard: heldStd,
		},
		AvailableInventory: InventoryBreakdown{
			VIP:      availVIP,
			Standard: availStd,
		},
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(metrics))
}

// GetHolds returns the active reservation holds queue.
func (h *AdminHandler) GetHolds(c *gin.Context) {
	ctx := c.Request.Context()

	rows, err := h.db.QueryContext(ctx, `
		SELECT id, ticket_code, category, price, session_id, expires_at 
		FROM tickets 
		WHERE status = 'Holding' 
		ORDER BY expires_at ASC
	`)
	if err != nil {
		logger.Error("Failed to query active holds", "error", err)
		c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
			errors.ErrCodeInternal,
			"An unexpected database error occurred",
			nil,
		))
		return
	}
	defer rows.Close()

	holds := []HoldDetail{}
	for rows.Next() {
		var hold HoldDetail
		var expiresAt time.Time
		err := rows.Scan(
			&hold.TicketID,
			&hold.TicketCode,
			&hold.Category,
			&hold.Price,
			&hold.SessionID,
			&expiresAt,
		)
		if err != nil {
			logger.Error("Failed to scan hold row", "error", err)
			c.JSON(http.StatusInternalServerError, types.NewErrorResponse(
				errors.ErrCodeInternal,
				"An unexpected database error occurred",
				nil,
			))
			return
		}

		hold.ExpiresAt = expiresAt.Format(time.RFC3339)
		secondsRemaining := int(time.Until(expiresAt).Seconds())
		if secondsRemaining < 0 {
			secondsRemaining = 0
		}
		hold.SecondsRemaining = secondsRemaining

		holds = append(holds, hold)
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(holds))
}

