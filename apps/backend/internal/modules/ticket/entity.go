package ticket

import "time"

// TicketStatus represents the lifecycle state of a ticket.
type TicketStatus string

const (
	StatusAvailable TicketStatus = "Available"
	StatusHolding   TicketStatus = "Holding"
	StatusSold      TicketStatus = "Sold"
)

// Ticket represents the ticket database entity.
type Ticket struct {
	ID         int64        `json:"id"`
	TicketCode string       `json:"ticket_code"`
	Category   string       `json:"category"` // VIP, Standard
	Price      float64      `json:"price"`
	Status     TicketStatus `json:"status"`
	SessionID  *string      `json:"session_id,omitempty"`
	HeldAt     *time.Time   `json:"held_at,omitempty"`
	ExpiresAt  *time.Time   `json:"expires_at,omitempty"`
}
