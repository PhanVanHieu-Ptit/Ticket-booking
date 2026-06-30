package utils

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// GenerateSecureToken generates a cryptographically secure random token of the specified byte length,
// returned as a hexadecimal string. The resulting string will be twice the length of the byte size.
func GenerateSecureToken(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// IsValidUUID checks if the given string is a valid UUID format.
func IsValidUUID(uuid string) bool {
	return uuidRegex.MatchString(uuid)
}
