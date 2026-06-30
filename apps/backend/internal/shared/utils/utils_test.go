package utils

import (
	"testing"
)

func TestGenerateSecureToken(t *testing.T) {
	token1, err := GenerateSecureToken(16)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Hex-encoded string has 2 characters per byte, so 16 bytes = 32 characters
	if len(token1) != 32 {
		t.Errorf("expected token length of 32, got %d", len(token1))
	}

	token2, err := GenerateSecureToken(16)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token1 == token2 {
		t.Error("expected tokens to be unique, got identical tokens")
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name string
		uuid string
		want bool
	}{
		{
			name: "valid v4 UUID",
			uuid: "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			want: true,
		},
		{
			name: "valid uppercase UUID",
			uuid: "F47AC10B-58CC-4372-A567-0E02B2C3D479",
			want: true,
		},
		{
			name: "invalid characters",
			uuid: "g47ac10b-58cc-4372-a567-0e02b2c3d479",
			want: false,
		},
		{
			name: "too short",
			uuid: "f47ac10b-58cc-4372-a567",
			want: false,
		},
		{
			name: "empty string",
			uuid: "",
			want: false,
		},
		{
			name: "missing hyphens",
			uuid: "f47ac10b58cc4372a5670e02b2c3d479",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidUUID(tt.uuid); got != tt.want {
				t.Errorf("IsValidUUID() = %v, want %v", got, tt.want)
			}
		})
	}
}
