package session

import "time"

// CreateSessionResponse represents the response after creating a user session.
type CreateSessionResponse struct {
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AdminLoginRequest represents the request payload for admin login.
type AdminLoginRequest struct {
	Passcode string `json:"passcode" binding:"required"`
}

// AdminLoginResponse represents the response payload for admin login.
type AdminLoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}
