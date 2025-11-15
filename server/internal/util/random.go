package util

import (
	"crypto/rand"
	"encoding/hex"
)

// RandHex returns n-byte random hex string.
func RandHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
