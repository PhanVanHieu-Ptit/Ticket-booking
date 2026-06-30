package session

import "time"

// UserSession represents an anonymous user session.
type UserSession struct {
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AdminSession represents an authenticated admin session.
type AdminSession struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
