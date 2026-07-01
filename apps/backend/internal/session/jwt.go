package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

// SessionClaims represents the JWT claims for a user session
type SessionClaims struct {
	jwt.RegisteredClaims
}

// AdminClaims represents the JWT claims for an admin session
type AdminClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// SignSessionToken generates a signed JWT session token for a given session ID (in the 'sub' claim).
// The token expires in 30 minutes.
func SignSessionToken(sessionID string, secret []byte) (string, time.Time, error) {
	expiresAt := time.Now().Add(30 * time.Minute)
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   sessionID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign session token: %w", err)
	}
	return tokenStr, expiresAt, nil
}

// VerifySessionToken parses and validates a JWT session token and returns the session ID and its expiration.
func VerifySessionToken(tokenStr string, secret []byte) (string, time.Time, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &SessionClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return "", time.Time{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(*SessionClaims)
	if !ok || !token.Valid {
		return "", time.Time{}, ErrInvalidToken
	}

	expiresAt := claims.ExpiresAt.Time
	return claims.Subject, expiresAt, nil
}

// SignAdminToken generates a signed JWT admin token.
// The token expires in 2 hours.
func SignAdminToken(secret []byte) (string, time.Time, error) {
	expiresAt := time.Now().Add(2 * time.Hour)
	claims := AdminClaims{
		Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "admin",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign admin token: %w", err)
	}
	return tokenStr, expiresAt, nil
}

// VerifyAdminToken parses and validates an admin JWT token and checks if role is 'admin'.
func VerifyAdminToken(tokenStr string, secret []byte) (bool, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &AdminClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return false, ErrInvalidToken
	}

	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return false, ErrInvalidToken
	}

	if claims.Role != "admin" {
		return false, nil
	}

	return true, nil
}
