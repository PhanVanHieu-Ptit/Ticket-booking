package admin

import "time"

// SalesMetrics represents the calculated sales performance data.
type SalesMetrics struct {
	TotalTicketsSold   int64            `json:"total_tickets_sold"`
	TotalRevenue       float64          `json:"total_revenue"`
	RemainingInventory map[string]int64 `json:"remaining_inventory"`
	HeldInventory      map[string]int64 `json:"held_inventory"`
	AvailableInventory map[string]int64 `json:"available_inventory"`
}

// ActiveHoldDetail represents a ticket currently held in a user session.
type ActiveHoldDetail struct {
	TicketID         int64     `json:"ticket_id"`
	TicketCode       string    `json:"ticket_code"`
	Category         string    `json:"category"`
	Price            float64   `json:"price"`
	SessionID        string    `json:"session_id"`
	HeldAt           time.Time `json:"held_at"`
	ExpiresAt        time.Time `json:"expires_at"`
	SecondsRemaining int64     `json:"seconds_remaining"`
}
