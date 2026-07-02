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

	// Ticket reservation specific error codes
	ErrCodeActiveHoldExists    = "ACTIVE_HOLD_EXISTS"
	ErrCodePurchaseLimitExceed = "PURCHASE_LIMIT_EXCEEDED"
	ErrCodeInvalidCategory     = "INVALID_CATEGORY"
	ErrCodeTicketUnavailable   = "TICKET_UNAVAILABLE"
	ErrCodeNoActiveHold        = "NO_ACTIVE_HOLD"
	ErrCodeReservationFailed   = "RESERVATION_FAILED"

	// Payment specific error codes
	ErrCodePaymentFailed = "PAYMENT_FAILED"

	// Idempotency specific error codes
	ErrCodeDuplicateRequest = "DUPLICATE_REQUEST"
)
