package session

import (
	"testing"
)

func TestSessionTokenSigningAndVerification(t *testing.T) {
	secret := []byte("test-jwt-secret-key-2026")
	sessionID := "sess_test_12345"

	tokenStr, exp, err := SignSessionToken(sessionID, secret)
	if err != nil {
		t.Fatalf("Failed to sign session token: %v", err)
	}

	if tokenStr == "" {
		t.Fatalf("Signed token is empty")
	}

	parsedID, parsedExp, err := VerifySessionToken(tokenStr, secret)
	if err != nil {
		t.Fatalf("Failed to verify session token: %v", err)
	}

	if parsedID != sessionID {
		t.Errorf("Expected session ID %q, got %q", sessionID, parsedID)
	}

	if parsedExp.Unix() != exp.Unix() {
		t.Errorf("Expected expiration %v, got %v", exp.Unix(), parsedExp.Unix())
	}
}

func TestAdminTokenSigningAndVerification(t *testing.T) {
	secret := []byte("test-jwt-secret-key-2026")

	tokenStr, _, err := SignAdminToken(secret)
	if err != nil {
		t.Fatalf("Failed to sign admin token: %v", err)
	}

	if tokenStr == "" {
		t.Fatalf("Signed admin token is empty")
	}

	valid, err := VerifyAdminToken(tokenStr, secret)
	if err != nil {
		t.Fatalf("Failed to verify admin token: %v", err)
	}

	if !valid {
		t.Errorf("Expected token to be valid admin token")
	}
}

func TestInvalidTokenVerification(t *testing.T) {
	secret := []byte("test-jwt-secret-key-2026")
	wrongSecret := []byte("wrong-secret-key")

	tokenStr, _, err := SignSessionToken("sess_123", secret)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	_, _, err = VerifySessionToken(tokenStr, wrongSecret)
	if err == nil {
		t.Errorf("Expected error verifying with wrong secret, got nil")
	}
}
