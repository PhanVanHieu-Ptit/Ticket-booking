package session

import (
	"context"
)

type sessionService struct {
	repo Repository
}

// NewSessionService creates a new instance of session Service.
func NewSessionService(repo Repository) Service {
	return &sessionService{
		repo: repo,
	}
}

func (s *sessionService) CreateUserSession(ctx context.Context) (*UserSession, error) {
	return &UserSession{}, nil
}

func (s *sessionService) ValidateUserSession(ctx context.Context, sessionID string) (bool, error) {
	return true, nil
}

func (s *sessionService) AdminLogin(ctx context.Context, passcode string) (*AdminSession, error) {
	return &AdminSession{}, nil
}

func (s *sessionService) ValidateAdminToken(ctx context.Context, token string) (bool, error) {
	return true, nil
}
