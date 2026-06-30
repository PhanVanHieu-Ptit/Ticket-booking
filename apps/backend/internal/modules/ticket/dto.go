package ticket

import "time"

// CategoryAvailability represents availability details for a single category.
type CategoryAvailability struct {
	Category  string  `json:"category"`
	Price     float64 `json:"price"`
	Available int     `json:"available"`
	Total     int     `json:"total"`
}

// AvailabilityResponse represents the response for ticket availability.
type AvailabilityResponse struct {
	Categories []CategoryAvailability `json:"categories"`
}

// ReserveRequest represents the request payload to reserve a ticket.
type ReserveRequest struct {
	Category string `json:"category" binding:"required"`
}

// ReserveResponse represents the response after a successful reservation.
type ReserveResponse struct {
	TicketID  int64     `json:"ticket_id"`
	Category  string    `json:"category"`
	ExpiresAt time.Time `json:"expires_at"`
}

// HoldResponse represents the response containing active hold details.
type HoldResponse struct {
	TicketID          int64     `json:"ticket_id"`
	Category          string    `json:"category"`
	Price             float64   `json:"price"`
	ExpiresAt         time.Time `json:"expires_at"`
	SecondsRemaining  int64     `json:"seconds_remaining"`
}
