package constants

import (
	"fmt"
	"time"
)

// Ticket Categories
const (
	TicketCategoryVIP      = "VIP"
	TicketCategoryStandard = "Standard"
)

// Ticket Statuses
const (
	TicketStatusAvailable = "Available"
	TicketStatusHolding   = "Holding"
	TicketStatusSold      = "Sold"
)

// Order/Transaction Statuses
const (
	OrderStatusPending   = "Pending"
	OrderStatusCompleted = "Completed"
	OrderStatusFailed    = "Failed"
	OrderStatusExpired   = "Expired"
)

// Redis Key Templates and Settings
const (
	// Redis key for holding a ticket session: "hold:{sessionID}" -> value: "{ticketID}:{ticketCategory}"
	RedisKeyHoldPrefix = "hold"

	// Redis set of available ticket IDs: "tickets:available:{category}"
	RedisKeyAvailableTicketsPrefix = "tickets:available"

	// Redis set of sessions that have purchased a ticket: "purchased:sessions"
	RedisKeyPurchasedSessions = "purchased:sessions"

	// Ticket Hold duration (5 minutes)
	TicketHoldTTL = 300 * time.Second
)

// RedisKeyTicketHold returns the Redis key for a ticket hold session.
func RedisKeyTicketHold(sessionID string) string {
	return fmt.Sprintf("%s:%s", RedisKeyHoldPrefix, sessionID)
}

// RedisKeyAvailableTickets returns the Redis key for available tickets in a category.
func RedisKeyAvailableTickets(category string) string {
	return fmt.Sprintf("%s:%s", RedisKeyAvailableTicketsPrefix, category)
}
