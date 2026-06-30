package validator

import (
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
)

type TestUser struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=18"`
}

func TestValidateStruct_Success(t *testing.T) {
	u := TestUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	if err := ValidateStruct(u); err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

func TestValidateStruct_Failure(t *testing.T) {
	u := TestUser{
		Name:  "", // Required violation
		Email: "invalid-email", // Email violation
		Age:   15, // Min violation
	}

	err := ValidateStruct(u)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected error to be of type *AppError, got %T", err)
	}

	if appErr.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", appErr.HTTPStatus)
	}

	if appErr.Code != apperrors.ErrCodeInvalidInput {
		t.Errorf("expected code %s, got %s", apperrors.ErrCodeInvalidInput, appErr.Code)
	}

	details, ok := appErr.Details.([]ValidationErrorDetail)
	if !ok {
		t.Fatalf("expected details to be []ValidationErrorDetail, got %T", appErr.Details)
	}

	if len(details) != 3 {
		t.Errorf("expected 3 validation errors, got %d", len(details))
	}

	// Verify details mapping (using JSON tag names due to RegisterTagNameFunc)
	fieldErrors := make(map[string]ValidationErrorDetail)
	for _, d := range details {
		fieldErrors[d.Field] = d
	}

	if _, ok := fieldErrors["name"]; !ok || fieldErrors["name"].Rule != "required" {
		t.Errorf("expected 'name' field to fail 'required' rule")
	}

	if _, ok := fieldErrors["email"]; !ok || fieldErrors["email"].Rule != "email" {
		t.Errorf("expected 'email' field to fail 'email' rule")
	}

	if _, ok := fieldErrors["age"]; !ok || fieldErrors["age"].Rule != "min" {
		t.Errorf("expected 'age' field to fail 'min' rule")
	}
}
