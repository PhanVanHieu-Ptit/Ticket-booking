package types

// ResponseEnvelope is the standard JSON wrapper for all API responses.
type ResponseEnvelope[T any] struct {
	Success bool      `json:"success"`
	Data    T         `json:"data,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// APIError represents the structured error body returned to the client.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
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

// NewErrorResponse creates an error response envelope.
func NewErrorResponse(code string, message string, details any) ResponseEnvelope[any] {
	return ResponseEnvelope[any]{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}
