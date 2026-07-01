package payment

import (
	"context"
	"database/sql"
)

type paymentRepository struct {
	db *sql.DB
}

// NewPaymentRepository creates a new instance of payment Repository.
func NewPaymentRepository(db *sql.DB) Repository {
	return &paymentRepository{
		db: db,
	}
}

func (r *paymentRepository) GetDB() *sql.DB {
	return r.db
}

func (r *paymentRepository) LockTicket(ctx context.Context, tx *sql.Tx, ticketID int64) (*Ticket, error) {
	var t Ticket
	var sessionID sql.NullString
	var expiresAt sql.NullTime

	err := tx.QueryRowContext(ctx, `
		SELECT id, ticket_code, category, price, status, session_id, expires_at 
		FROM tickets 
		WHERE id = $1 
		FOR UPDATE
	`, ticketID).Scan(&t.ID, &t.TicketCode, &t.Category, &t.Price, &t.Status, &sessionID, &expiresAt)

	if err != nil {
		return nil, err
	}

	if sessionID.Valid {
		t.SessionID = &sessionID.String
	}
	if expiresAt.Valid {
		t.ExpiresAt = &expiresAt.Time
	}

	return &t, nil
}

func (r *paymentRepository) UpdateTicketStatus(ctx context.Context, tx *sql.Tx, ticketID int64, status string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE tickets 
		SET status = $2, updated_at = NOW() 
		WHERE id = $1
	`, ticketID, status)
	return err
}

func (r *paymentRepository) CreateOrder(ctx context.Context, tx *sql.Tx, order *Order) error {
	err := tx.QueryRowContext(ctx, `
		INSERT INTO orders (ticket_id, session_id, amount, status, email, card_holder_name, payment_reference)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, order.TicketID, order.SessionID, order.Amount, order.Status, order.Email, order.CardHolderName, order.PaymentReference).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	return err
}
