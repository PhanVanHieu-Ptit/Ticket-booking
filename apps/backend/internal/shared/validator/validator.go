package validator

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	apperrors "github.com/PhanVanHieu-Ptit/ticket-booking/backend/internal/shared/errors"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Use JSON tag name for validation errors instead of struct field name
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// ValidationErrorDetail represents the details of a single field validation failure.
type ValidationErrorDetail struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
	Param string `json:"param,omitempty"`
	Value any    `json:"value,omitempty"`
}

// ValidateStruct validates a struct and returns an AppError if validation fails.
func ValidateStruct(s interface{}) error {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		details := make([]ValidationErrorDetail, len(valErrs))
		for i, ve := range valErrs {
			details[i] = ValidationErrorDetail{
				Field: ve.Field(),
				Rule:  ve.Tag(),
				Param: ve.Param(),
				Value: ve.Value(),
			}
		}

		return &apperrors.AppError{
			HTTPStatus: http.StatusBadRequest,
			Code:       apperrors.ErrCodeInvalidInput,
			Message:    "Validation failed for one or more fields",
			Details:    details,
		}
	}

	// For other validation errors (e.g. invalid validation input)
	return apperrors.NewInternal(err, "Failed to perform struct validation")
}

// CustomRule registers a custom validation rule.
func CustomRule(tag string, fn validator.Func) error {
	return validate.RegisterValidation(tag, fn)
}

// GetFieldErrorMessage provides a simple human-readable message for standard validation rules.
func GetFieldErrorMessage(err ValidationErrorDetail) string {
	switch err.Rule {
	case "required":
		return fmt.Sprintf("The %s field is required", err.Field)
	case "email":
		return fmt.Sprintf("The %s field must be a valid email address", err.Field)
	case "min":
		return fmt.Sprintf("The %s field must be at least %s", err.Field, err.Param)
	case "max":
		return fmt.Sprintf("The %s field cannot be greater than %s", err.Field, err.Param)
	default:
		return fmt.Sprintf("The %s field failed validation on rule '%s'", err.Field, err.Rule)
	}
}
