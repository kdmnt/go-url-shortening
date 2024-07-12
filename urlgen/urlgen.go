// Package urlgen provides utility functions for the URL shortener service.
package urlgen

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// charset defines the character set used for generating short URLs.
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// shortURLLength defines the length of the generated short URLs.
const shortURLLength = 8

// Generate creates a new short URL string.
func Generate() (string, error) {
	var sb strings.Builder
	sb.Grow(shortURLLength) // Pre-allocate the required capacity for better performance

	charsetLength := big.NewInt(int64(len(charset)))

	for i := 0; i < shortURLLength; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		sb.WriteByte(charset[randomIndex.Int64()])
	}
	return sb.String(), nil
}
