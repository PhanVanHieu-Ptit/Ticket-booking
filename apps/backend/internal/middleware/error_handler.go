package middleware

import (
	"log/slog"
	"net/http"

	apperrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/httpx"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware is the single global exception filter: every
// handler and middleware downstream reports failures via c.Error(err)
// instead of writing its own JSON, and this is the only place that
// serializes an error response. isProd controls whether 5xx internal
// details are redacted from the response body.
func ErrorHandlerMiddleware(isProd bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		appErr := apperrors.FromError(c.Errors.Last().Err)
		ctx := c.Request.Context()

		if appErr.HTTPStatus >= http.StatusInternalServerError {
			logger.ErrorContext(ctx, "Unhandled request error",
				slog.String("code", appErr.Code),
				slog.String("path", c.Request.URL.Path),
				slog.Any("error", appErr.Unwrap()),
			)
		} else {
			logger.WarnContext(ctx, "Request failed",
				slog.String("code", appErr.Code),
				slog.String("path", c.Request.URL.Path),
			)
		}

		httpx.RespondError(c, appErr, isProd)
	}
}
