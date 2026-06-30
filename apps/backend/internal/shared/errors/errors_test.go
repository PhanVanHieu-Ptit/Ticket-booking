package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	err := New(http.StatusNotFound, ErrCodeNotFound, "Resource not found")
	expected := "[NOT_FOUND] Resource not found"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := errors.New("db connection failed")
	err := NewInternal(cause, "Internal database error")

	if !errors.Is(err, cause) {
		t.Errorf("expected wrapped error to be cause")
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Errorf("expected unwrapped error to be %v, got %v", cause, unwrapped)
	}
}

func TestFromError(t *testing.T) {
	// Case 1: Nil error
	if FromError(nil) != nil {
		t.Error("expected nil for nil input")
	}

	// Case 2: Already an AppError
	appErr := New(http.StatusConflict, ErrCodeConflict, "Conflict occurred")
	converted := FromError(appErr)
	if converted != appErr {
		t.Error("expected original AppError to be returned directly")
	}

	// Case 3: Standard Go error
	stdErr := errors.New("standard error")
	converted2 := FromError(stdErr)
	if converted2.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("expected 500 status, got %d", converted2.HTTPStatus)
	}
	if converted2.Code != ErrCodeInternal {
		t.Errorf("expected %s code, got %s", ErrCodeInternal, converted2.Code)
	}
	if converted2.Internal != stdErr {
		t.Errorf("expected internal error to be wrapped")
	}
}
