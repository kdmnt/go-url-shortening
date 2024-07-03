// Package types defines the data structures used in the URL shortener service.
package types

import "time"

// URLResponse represents the response structure for URL-related operations.
type URLResponse struct {
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// URLData represents the internal structure for storing URL data.
type URLData struct {
	ShortURL    string
	OriginalURL string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// URLRequest represents the request structure for creating or updating a short URL.
type URLRequest struct {
	URL string `json:"url" validate:"required,url"`
}
