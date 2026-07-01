package errors

// Standard machine-readable error codes
const (
	ErrCodeInternal      = "INTERNAL_ERROR"
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeUnauthorized  = "UNAUTHORIZED"
	ErrCodeForbidden     = "FORBIDDEN"
	ErrCodeInvalidInput  = "INVALID_INPUT"
	ErrCodeConflict      = "CONFLICT"
	ErrCodeTicketSoldOut = "TICKET_SOLD_OUT"
	ErrCodeHoldExpired   = "HOLD_EXPIRED"
	ErrCodeLimitExceeded = "LIMIT_EXCEEDED"
	ErrCodeSessionRequired = "SESSION_REQUIRED"
	ErrCodeAdminUnauthorized = "ADMIN_UNAUTHORIZED"
	ErrCodeInvalidSessionToken = "INVALID_SESSION_TOKEN"
)
