package middleware

import (
	"net/http"
	"strings"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/session"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/gin-gonic/gin"
)

// AdminAuthMiddleware verifies the admin JWT token in the Authorization header.
func AdminAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Error(errors.New(http.StatusUnauthorized, errors.ErrCodeAdminUnauthorized, "Admin authentication token is required"))
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.Error(errors.New(http.StatusUnauthorized, errors.ErrCodeAdminUnauthorized, "Invalid authorization header format. Expected: Bearer <token>"))
			c.Abort()
			return
		}

		tokenStr := parts[1]
		valid, err := session.VerifyAdminToken(tokenStr, []byte(jwtSecret))
		if err != nil || !valid {
			c.Error(errors.New(http.StatusUnauthorized, errors.ErrCodeAdminUnauthorized, "Invalid or expired admin session token"))
			c.Abort()
			return
		}

		c.Next()
	}
}
