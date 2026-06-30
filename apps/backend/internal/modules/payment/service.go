package payment

import (
	"context"
)

type paymentService struct {
	repo Repository
}

// NewPaymentService creates a new instance of payment Service.
func NewPaymentService(repo Repository) Service {
	return &paymentService{
		repo: repo,
	}
}

func (s *paymentService) Checkout(ctx context.Context, sessionID string, ticketID int64, email string, cardHolderName string, simulateStatus string) (*Order, error) {
	return &Order{}, nil
}
