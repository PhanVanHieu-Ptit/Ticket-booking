package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"time"

	apperrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/httpx"
	"github.com/gin-gonic/gin"
)

var log *slog.Logger

// Init initializes the global structured logger.
func Init(env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	if env == "development" || env == "" {
		opts.Level = slog.LevelDebug
		// Text handler for readable local development console output
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// JSON handler for production logging systems (ELK, Loki, CloudWatch)
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	log = slog.New(handler)
	slog.SetDefault(log)
}

// Get returns the global logger instance.
func Get() *slog.Logger {
	if log == nil {
		// Fallback if not initialized
		Init("development")
	}
	return log
}

// GinMiddleware returns a Gin middleware that logs HTTP requests using the structured logger.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		// Determine log level based on status code
		var logFn func(string, ...any)
		logger := Get()

		if status >= 500 {
			logFn = logger.Error
		} else if status >= 400 {
			logFn = logger.Warn
		} else {
			logFn = logger.Info
		}

		ctx := c.Request.Context()

		// Include request metadata in the log
		logFn("HTTP Request",
			slog.String("method", method),
			slog.String("path", path),
			slog.String("query", query),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("ip", clientIP),
			slog.String("user_agent", c.Request.UserAgent()),
		)

		// Ensure errors attached to Gin context are logged
		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				logger.ErrorContext(ctx, "Gin context error",
					slog.String("err", e.Error()),
					slog.String("path", path),
				)
			}
		}
	}
}

// RecoveryMiddleware returns a Gin middleware that recovers from any panics, logs the stack trace,
// and returns a standardized 500 error response via the shared httpx.RespondError writer, keeping
// the panic path identical in shape to every other error response. isProd controls whether the
// panic's internal details are redacted from the response body.
func RecoveryMiddleware(isProd bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection (broken pipe or connection reset)
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					var se *os.SyscallError
					if errors.As(ne.Err, &se) {
						errStr := strings.ToLower(se.Error())
						if strings.Contains(errStr, "broken pipe") || strings.Contains(errStr, "connection reset") {
							brokenPipe = true
						}
					}
				}

				logger := Get()
				ctx := c.Request.Context()

				if brokenPipe {
					logger.ErrorContext(ctx, "Broken pipe / connection reset by peer",
						slog.Any("error", err),
					)
					c.Abort()
					return
				}

				// Log panic with stack trace
				logger.ErrorContext(ctx, "Recovery from panic",
					slog.Any("error", err),
					slog.String("stack", string(debug.Stack())),
				)

				appErr := apperrors.NewInternal(fmt.Errorf("panic: %v", err), "An unexpected error occurred")
				c.Abort()
				httpx.RespondError(c, appErr, isProd)
			}
		}()
		c.Next()
	}
}

// Info logs at info level.
func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

// Debug logs at debug level.
func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

// Warn logs at warn level.
func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

// Error logs at error level.
func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

// InfoContext logs at info level with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	Get().InfoContext(ctx, msg, args...)
}

// DebugContext logs at debug level with context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	Get().DebugContext(ctx, msg, args...)
}

// WarnContext logs at warn level with context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	Get().WarnContext(ctx, msg, args...)
}

// ErrorContext logs at error level with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	Get().ErrorContext(ctx, msg, args...)
}
