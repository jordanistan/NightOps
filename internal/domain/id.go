package domain

import (
	"crypto/rand"
	"encoding/hex"
)

// NewID returns a stable random identifier suitable for a local aggregate.
func NewID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
