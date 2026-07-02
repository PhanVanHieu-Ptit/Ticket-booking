// Package httpx holds the single, shared writer for error HTTP responses so
// every error - validation, DB, business logic, or panic - is serialized the
// same way regardless of which middleware or handler produced it.
package httpx

import (
	"net/http"
	"time"

	apperrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/types"
	"github.com/gin-gonic/gin"
)

// RespondError writes the standard error envelope for appErr onto c.
// When redactInternal is true, 5xx errors have their message and details
// replaced with a generic, safe message so internal details (raw DB errors,
// wrapped Go errors, etc.) are never leaked to the client in production.
func RespondError(c *gin.Context, appErr *apperrors.AppError, redactInternal bool) {
	message := appErr.Message
	details := appErr.Details

	if redactInternal && appErr.HTTPStatus >= http.StatusInternalServerError {
		message = "An unexpected error occurred"
		details = nil
	}

	c.JSON(appErr.HTTPStatus, types.ResponseEnvelope[any]{
		Success: false,
		Error: &types.APIError{
			Code:       appErr.Code,
			Message:    message,
			StatusCode: appErr.HTTPStatus,
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Path:       c.Request.URL.Path,
			Details:    details,
		},
	})
}
