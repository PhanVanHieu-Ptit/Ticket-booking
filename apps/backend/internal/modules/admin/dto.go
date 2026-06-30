package admin

// MetricsResponse represents the response for GET /api/v1/admin/metrics
type MetricsResponse struct {
	TotalTicketsSold   int64            `json:"total_tickets_sold"`
	TotalRevenue       float64          `json:"total_revenue"`
	RemainingInventory map[string]int64 `json:"remaining_inventory"`
	HeldInventory      map[string]int64 `json:"held_inventory"`
	AvailableInventory map[string]int64 `json:"available_inventory"`
}

// HoldsResponse represents the response for GET /api/v1/admin/holds
type HoldsResponse struct {
	Holds []ActiveHoldDetail `json:"holds"`
}
