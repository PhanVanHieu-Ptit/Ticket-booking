package session

import (
	"context"
)

// Repository defines the data store operations for sessions.
type Repository interface {
	GetAdminPasscodeHash(ctx context.Context) (string, error)
}

// Service defines the business logic operations for sessions.
type Service interface {
	CreateUserSession(ctx context.Context) (*UserSession, error)
	ValidateUserSession(ctx context.Context, sessionID string) (bool, error)
	AdminLogin(ctx context.Context, passcode string) (*AdminSession, error)
	ValidateAdminToken(ctx context.Context, token string) (bool, error)
}
