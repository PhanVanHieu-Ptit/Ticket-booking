package types

// ResponseEnvelope is the standard JSON wrapper for all API responses.
type ResponseEnvelope[T any] struct {
	Success bool      `json:"success"`
	Data    T         `json:"data"`
	Error   *APIError `json:"error,omitempty"`
}

// APIError represents the structured error body returned to the client.
// Code and Message are kept at their existing paths for frontend backward
// compatibility; StatusCode/Timestamp/Path are additive fields populated by
// the global error handler so every error response carries the same shape.
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
	Timestamp  string `json:"timestamp"`
	Path       string `json:"path"`
	Details    any    `json:"details,omitempty"`
}

// PaginationMetadata contains pagination details for list responses.
type PaginationMetadata struct {
	TotalCount int `json:"total_count"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

// NewSuccessResponse creates a successful response envelope.
func NewSuccessResponse[T any](data T) ResponseEnvelope[T] {
	return ResponseEnvelope[T]{
		Success: true,
		Data:    data,
	}
}

