package ticket

import (
	"context"
)

type ticketService struct {
	repo Repository
}

// NewTicketService creates a new instance of ticket Service.
func NewTicketService(repo Repository) Service {
	return &ticketService{
		repo: repo,
	}
}

func (s *ticketService) GetAvailability(ctx context.Context) ([]Ticket, error) {
	return s.repo.GetAvailability(ctx)
}

func (s *ticketService) GetActiveHold(ctx context.Context, sessionID string) (*Ticket, error) {
	return s.repo.GetActiveHold(ctx, sessionID)
}

func (s *ticketService) ReserveTicket(ctx context.Context, sessionID string, category string) (*Ticket, error) {
	return s.repo.ReserveTicket(ctx, sessionID, category)
}

func (s *ticketService) CancelHold(ctx context.Context, sessionID string) error {
	return s.repo.CancelHold(ctx, sessionID)
}
