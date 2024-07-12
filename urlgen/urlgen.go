// Package urlgen provides utility functions for the URL shortener service.
package urlgen

import (
	"crypto/rand"
	"math/big"
)

// charset defines the character set used for generating short URLs.
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// shortURLLength defines the length of the generated short URLs.
const shortURLLength = 8

// Generate creates a new short URL string.
func Generate() (string, error) {
	shortURL := make([]byte, shortURLLength)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := range shortURL {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", err
		}
		shortURL[i] = charset[randomIndex.Int64()]
	}
	return string(shortURL), nil
}