// Package storage provides interfaces and common errors for URL storage operations.
package storage

import (
	"context"
	"errors"
	"go-url-shortening/types"
)

// Common errors returned by storage operations.
var (
	ErrShortURLExists         = errors.New("short URL already exists")
	ErrShortURLNotFound       = errors.New("short URL not found")
	ErrStorageCapacityReached = errors.New("storage capacity reached")
)

// Storage interface defines the methods for URL storage operations.
type Storage interface {
	Create(ctx context.Context, urlData types.URLData) error
	GetURLData(ctx context.Context, shortURL string) (types.URLData, error)
	GetShortURL(ctx context.Context, originalURL string) (string, error)
	Update(ctx context.Context, urlData types.URLData) error
	Delete(ctx context.Context, shortURL string) error
}
