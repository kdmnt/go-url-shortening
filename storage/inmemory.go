package storage

import (
	"context"
	"sync"
	"time"

	"go-url-shortening/types"
	"go.uber.org/zap"
)

// InMemoryStorage implements the Storage interface using an in-memory map.
type InMemoryStorage struct {
	urls     map[string]types.URLData // Map to store short URL to URLData mappings
	mu       sync.RWMutex             // Read-write mutex for thread-safe access to the map
	capacity int                      // Maximum number of URLs that can be stored
	count    int                      // Current number of stored URLs
	logger   *zap.Logger              // Logger for InMemoryStorage operations
}

// The sync.RWMutex (mu) is used to ensure thread-safe access to the shared resources (urls and count).
// It allows multiple readers to access the data simultaneously, but ensures exclusive access for writers.
// This is particularly useful for operations that only read data (like Read) to proceed concurrently,
// while write operations (like Create, Update, and Delete) have exclusive access.

// Note: URL validation is performed at the handler level, not in the storage layer.
// This design decision allows for more flexibility in URL handling and validation.

// NewInMemoryStorage creates and returns a new InMemoryStorage instance
func NewInMemoryStorage(capacity int, logger *zap.Logger) *InMemoryStorage {
	if capacity <= 0 {
		capacity = 1000 // Default capacity if an invalid value is provided
	}
	if logger == nil {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			panic("Failed to initialize zap logger: " + err.Error())
		}
	}
	return &InMemoryStorage{
		urls:     make(map[string]types.URLData, capacity), // pre-allocates the map with the given capacity,
		capacity: capacity,                                 // can improve performance by reducing dynamic resizing
		logger:   logger,
	}
}

// Note: This is an in-memory implementation. For production use,
// consider implementing a persistent storage solution (e.g., database)
// by creating a new struct that implements the Storage interface.

// Create adds a new short URL and its corresponding URLData to the storage
func (s *InMemoryStorage) Create(ctx context.Context, urlData types.URLData) error {
	select {
	case <-ctx.Done():
		s.logger.Warn("Create operation cancelled", zap.String("shortURL", urlData.ShortURL))
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if s.count >= s.capacity {
			s.logger.Error("Storage capacity reached. Cannot create shortURL", zap.String("shortURL", urlData.ShortURL))
			return ErrStorageCapacityReached
		}
		if _, exists := s.urls[urlData.ShortURL]; exists {
			s.logger.Warn("Attempt to create duplicate shortURL", zap.String("shortURL", urlData.ShortURL))
			return ErrShortURLExists
		}

		urlData.CreatedAt = time.Now().UTC()
		urlData.UpdatedAt = urlData.CreatedAt
		s.urls[urlData.ShortURL] = urlData
		s.count++
		s.logger.Info("Short URL created successfully",
			zap.String("shortURL", urlData.ShortURL),
			zap.String("originalURL", urlData.OriginalURL),
			zap.Time("createdAt", urlData.CreatedAt))
		return nil
	}
}

// GetURLData retrieves the URLData for a given short URL.
func (s *InMemoryStorage) GetURLData(ctx context.Context, shortURL string) (types.URLData, error) {
	select {
	case <-ctx.Done():
		s.logger.Warn("Read operation cancelled", zap.String("shortURL", shortURL))
		return types.URLData{}, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		if urlData, exists := s.urls[shortURL]; exists {
			s.logger.Info("URL data retrieved successfully",
				zap.String("shortURL", shortURL),
				zap.String("originalURL", urlData.OriginalURL))
			return urlData, nil
		}
		return types.URLData{}, ErrShortURLNotFound
	}
}

// GetShortURL retrieves the short URL for a given original URL.
func (s *InMemoryStorage) GetShortURL(ctx context.Context, originalURL string) (string, error) {
	select {
	case <-ctx.Done():
		s.logger.Warn("GetShortURL operation cancelled", zap.String("originalURL", originalURL))
		return "", ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		for shortURL, storedOriginalURL := range s.urls {
			if storedOriginalURL.OriginalURL == originalURL {
				s.logger.Debug("Short URL retrieved successfully",
					zap.String("shortURL", shortURL),
					zap.String("originalURL", originalURL))
				return shortURL, nil
			}
		}
		return "", ErrShortURLNotFound
	}
}

// Update modifies the URLData for a given short URL.
func (s *InMemoryStorage) Update(ctx context.Context, urlData types.URLData) error {
	select {
	case <-ctx.Done():
		s.logger.Warn("Update operation cancelled", zap.String("shortURL", urlData.ShortURL))
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.urls[urlData.ShortURL]; !exists {
			s.logger.Warn("Attempt to update non-existent shortURL", zap.String("shortURL", urlData.ShortURL))
			return ErrShortURLNotFound
		}

		oldURLData := s.urls[urlData.ShortURL]
		urlData.CreatedAt = oldURLData.CreatedAt
		urlData.UpdatedAt = time.Now().UTC()
		s.urls[urlData.ShortURL] = urlData
		s.logger.Info("Updated shortURL",
			zap.String("shortURL", urlData.ShortURL),
			zap.String("oldURL", oldURLData.OriginalURL),
			zap.String("newURL", urlData.OriginalURL),
			zap.Time("updatedAt", urlData.UpdatedAt))
		return nil
	}
}

// Delete removes a short URL and its corresponding original URL from the storage.
func (s *InMemoryStorage) Delete(ctx context.Context, shortURL string) error {
	select {
	case <-ctx.Done():
		s.logger.Warn("Delete operation cancelled", zap.String("shortURL", shortURL))
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.urls[shortURL]; !exists {
			s.logger.Warn("Attempt to delete non-existent shortURL", zap.String("shortURL", shortURL))
			return ErrShortURLNotFound
		}

		delete(s.urls, shortURL)
		s.count--
		s.logger.Info("Deleted shortURL", zap.String("shortURL", shortURL))
		return nil
	}
}
