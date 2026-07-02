package payment

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/sse"
	appErrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

// duplicatePaidOrderConstraint is the DB-level safety net (see migration
// 000002) that rejects a second 'Paid' order for the same ticket even if
// application-level locking is ever bypassed.
const duplicatePaidOrderConstraint = "idx_orders_ticket_id_paid"

type paymentService struct {
	repo Repository
	rdb  *redis.Client
}

// NewPaymentService creates a new instance of payment Service.
func NewPaymentService(repo Repository, rdb *redis.Client) Service {
	return &paymentService{
		repo: repo,
		rdb:  rdb,
	}
}

func (s *paymentService) Checkout(ctx context.Context, sessionID string, ticketID int64, email string, cardHolderName string, simulateStatus string) (*Order, error) {
	db := s.repo.GetDB()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, appErrors.Wrap(err, appErrors.ErrCodeReservationFailed, "Failed to start database transaction", http.StatusInternalServerError)
	}
	defer tx.Rollback()

	// 1. Lock the ticket row
	ticket, err := s.repo.LockTicket(ctx, tx, ticketID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.New(http.StatusNotFound, appErrors.ErrCodeNotFound, "Ticket not found")
		}
		return nil, appErrors.NewInternal(err, "Failed to retrieve ticket")
	}

	// 2. Validate hold status
	if ticket.Status != "Holding" {
		return nil, appErrors.New(http.StatusGone, appErrors.ErrCodeHoldExpired, "Ticket hold is invalid or has expired")
	}

	// 3. Validate session ID
	if ticket.SessionID == nil || *ticket.SessionID != sessionID {
		return nil, appErrors.New(http.StatusForbidden, appErrors.ErrCodeForbidden, "This reservation does not belong to your session")
	}

	// 4. Validate hold has not expired
	if ticket.ExpiresAt == nil || ticket.ExpiresAt.Before(time.Now()) {
		return nil, appErrors.New(http.StatusGone, appErrors.ErrCodeHoldExpired, "Your reservation has expired")
	}

	// 5. Simulate payment gateway
	if simulateStatus == "fail" {
		return nil, appErrors.New(http.StatusPaymentRequired, "PAYMENT_FAILED", "Payment processing simulated failure")
	}

	// 6. Transition ticket status to Sold
	err = s.repo.UpdateTicketStatus(ctx, tx, ticketID, "Sold")
	if err != nil {
		return nil, appErrors.NewInternal(err, "Failed to update ticket status")
	}

	// 7. Create and record the order
	paymentRef := "PAY-" + uuid.New().String()
	order := &Order{
		TicketID:         ticketID,
		SessionID:        sessionID,
		Amount:           ticket.Price,
		Status:           StatusPaid,
		Email:            email,
		CardHolderName:   cardHolderName,
		PaymentReference: paymentRef,
	}

	err = s.repo.CreateOrder(ctx, tx, order)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == duplicatePaidOrderConstraint {
			return nil, appErrors.New(http.StatusConflict, appErrors.ErrCodeConflict, "This ticket has already been purchased")
		}
		return nil, appErrors.NewInternal(err, "Failed to record order")
	}

	// 8. Commit database transaction
	err = tx.Commit()
	if err != nil {
		return nil, appErrors.NewInternal(err, "Failed to commit checkout transaction")
	}

	// 9. Update Redis Concurrency Shield
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, "hold:"+sessionID)
	pipe.SAdd(ctx, "purchased:sessions", sessionID)
	_, redisErr := pipe.Exec(ctx)
	if redisErr != nil {
		logger.Error("Failed to update Redis shield on successful checkout", "error", redisErr, "session_id", sessionID)
	}

	// 10. Broadcast updated available count via SSE
	if available, err := s.rdb.SCard(ctx, "tickets:available:"+ticket.Category).Result(); err == nil {
		sse.BroadcastInventoryUpdate(ticket.Category, available)
	}

	return order, nil
}
