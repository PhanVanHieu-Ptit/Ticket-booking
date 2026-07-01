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
