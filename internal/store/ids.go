package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func NewID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func BoolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
