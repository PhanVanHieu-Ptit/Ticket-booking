package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError represents an application-specific error.
type AppError struct {
	HTTPStatus int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`          // Safe, user-facing error message
	Internal   error  `json:"-"`                // Raw system error (logged, not sent to client)
	Details    any    `json:"details,omitempty"` // Optional structured details (e.g., validation errors)
}

// Error implements the standard error interface.
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying wrapped error.
func (e *AppError) Unwrap() error {
	return e.Internal
}

// New creates a new AppError with the specified HTTP status, machine-readable code, and user message.
func New(httpStatus int, code string, message string) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
	}
}

// NewWithDetails creates an AppError with additional structured details.
func NewWithDetails(httpStatus int, code string, message string, details any) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
		Details:    details,
	}
}

// NewInternal creates an AppError for an internal server error.
// The raw error is wrapped and kept private.
func NewInternal(err error, message string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusInternalServerError,
		Code:       ErrCodeInternal,
		Message:    message,
		Internal:   err,
	}
}

// Wrap wraps an existing error into an AppError with a specific code and message.
func Wrap(err error, code string, message string, httpStatus int) *AppError {
	return &AppError{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
		Internal:   err,
	}
}

// Common sentinel errors
var (
	ErrNotFound = &AppError{
		HTTPStatus: http.StatusNotFound,
		Code:       ErrCodeNotFound,
		Message:    "Resource not found",
	}

	ErrUnauthorized = &AppError{
		HTTPStatus: http.StatusUnauthorized,
		Code:       ErrCodeUnauthorized,
		Message:    "Authentication required",
	}

	ErrForbidden = &AppError{
		HTTPStatus: http.StatusForbidden,
		Code:       ErrCodeForbidden,
		Message:    "Access denied",
	}
)

// FromError converts a generic Go error into an AppError.
// If the error is already an AppError, it is returned directly.
// If it's a standard error, it is wrapped as an internal server error.
func FromError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return &AppError{
		HTTPStatus: http.StatusInternalServerError,
		Code:       ErrCodeInternal,
		Message:    "An unexpected error occurred",
		Internal:   err,
	}
}
