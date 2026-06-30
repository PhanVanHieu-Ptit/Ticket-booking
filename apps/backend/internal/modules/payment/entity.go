package payment

import "time"

// OrderStatus represents the state of a payment order.
type OrderStatus string

const (
	StatusPaid     OrderStatus = "Paid"
	StatusRefunded OrderStatus = "Refunded"
	StatusFailed   OrderStatus = "Failed"
)

// Order represents the order database entity.
type Order struct {
	ID               string      `json:"id"`
	TicketID         int64       `json:"ticket_id"`
	SessionID        string      `json:"session_id"`
	Amount           float64     `json:"amount"`
	Status           OrderStatus `json:"status"`
	Email            string      `json:"email"`
	CardHolderName   string      `json:"card_holder_name"`
	PaymentReference string      `json:"payment_reference"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}
