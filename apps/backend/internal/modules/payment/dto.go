package payment

import "time"

// CheckoutRequest represents the request payload to complete a purchase.
type CheckoutRequest struct {
	TicketID       int64  `json:"ticket_id" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	CardHolderName string `json:"card_holder_name" binding:"required"`
	PaymentMethod  string `json:"payment_method" binding:"required"`
	SimulateStatus string `json:"simulate_status" binding:"required"` // "success" or "fail"
}

// CheckoutResponse represents the response after a successful checkout.
type CheckoutResponse struct {
	OrderID          string    `json:"order_id"`
	TicketID         int64     `json:"ticket_id"`
	Amount           float64   `json:"amount"`
	PaymentReference string    `json:"payment_reference"`
	PaidAt           time.Time `json:"paid_at"`
}
