package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// RandomGenerator generates cryptographically secure random strings
// using crypto/rand as required by the security specification.
type RandomGenerator struct{}

// Generate generates a base64url-encoded random string from the given number of bytes.
// The minimum recommended byteLen is 16 (128 bits of entropy).
func (g RandomGenerator) Generate(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
