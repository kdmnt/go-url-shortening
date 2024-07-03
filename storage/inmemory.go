package storage

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go-url-shortening/types"
)

// InMemoryStorage implements the Storage interface using an in-memory map.
type InMemoryStorage struct {
	urls     map[string]types.URLData // Map to store short URL to URLData mappings
	mu       sync.RWMutex             // Read-write mutex for thread-safe access to the map
	capacity int                      // Maximum number of URLs that can be stored
	count    int                      // Current number of stored URLs
	logger   *logrus.Logger           // Logger for InMemoryStorage operations
}

// The sync.RWMutex (mu) is used to ensure thread-safe access to the shared resources (urls and count).
// It allows multiple readers to access the data simultaneously, but ensures exclusive access for writers.
// This is particularly useful for operations that only read data (like Read) to proceed concurrently,
// while write operations (like Create, Update, and Delete) have exclusive access.

// Note: URL validation is performed at the handler level, not in the storage layer.
// This design decision allows for more flexibility in URL handling and validation.

// NewInMemoryStorage creates and returns a new InMemoryStorage instance
func NewInMemoryStorage(capacity int, logger *logrus.Logger) *InMemoryStorage {
	if capacity <= 0 {
		capacity = 1000 // Default capacity if an invalid value is provided
	}
	if logger == nil {
		logger = logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetLevel(logrus.InfoLevel)
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
		s.logger.WithField("shortURL", urlData.ShortURL).Warn("Create operation cancelled")
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if s.count >= s.capacity {
			s.logger.WithField("shortURL", urlData.ShortURL).Error("Storage capacity reached. Cannot create shortURL")
			return ErrStorageCapacityReached
		}
		if _, exists := s.urls[urlData.ShortURL]; exists {
			s.logger.WithField("shortURL", urlData.ShortURL).Warn("Attempt to create duplicate shortURL")
			return ErrShortURLExists
		}

		urlData.CreatedAt = time.Now()
		urlData.UpdatedAt = urlData.CreatedAt
		s.urls[urlData.ShortURL] = urlData
		s.count++
		s.logger.WithFields(logrus.Fields{
			"shortURL":    urlData.ShortURL,
			"originalURL": urlData.OriginalURL,
			"createdAt":   urlData.CreatedAt,
		}).Info("Short URL created successfully")
		return nil
	}
}

// GetURLData retrieves the URLData for a given short URL.
func (s *InMemoryStorage) GetURLData(ctx context.Context, shortURL string) (types.URLData, error) {
	select {
	case <-ctx.Done():
		s.logger.WithField("shortURL", shortURL).Warn("Read operation cancelled")
		return types.URLData{}, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		if urlData, exists := s.urls[shortURL]; exists {
			s.logger.WithFields(logrus.Fields{
				"shortURL":    shortURL,
				"originalURL": urlData.OriginalURL,
			}).Info("URL data retrieved successfully")
			return urlData, nil
		}
		return types.URLData{}, ErrShortURLNotFound
	}
}

// GetShortURL retrieves the short URL for a given original URL.
func (s *InMemoryStorage) GetShortURL(ctx context.Context, originalURL string) (string, error) {
	select {
	case <-ctx.Done():
		s.logger.WithField("originalURL", originalURL).Warn("GetShortURL operation cancelled")
		return "", ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()

		for shortURL, storedOriginalURL := range s.urls {
			if storedOriginalURL.OriginalURL == originalURL {
				s.logger.WithFields(logrus.Fields{
					"shortURL":    shortURL,
					"originalURL": originalURL,
				}).Debug("Short URL retrieved successfully")
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
		s.logger.WithField("shortURL", urlData.ShortURL).Warn("Update operation cancelled")
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.urls[urlData.ShortURL]; !exists {
			s.logger.WithField("shortURL", urlData.ShortURL).Warn("Attempt to update non-existent shortURL")
			return ErrShortURLNotFound
		}

		oldURLData := s.urls[urlData.ShortURL]
		urlData.CreatedAt = oldURLData.CreatedAt
		urlData.UpdatedAt = time.Now()
		s.urls[urlData.ShortURL] = urlData
		s.logger.WithFields(logrus.Fields{
			"shortURL":  urlData.ShortURL,
			"oldURL":    oldURLData.OriginalURL,
			"newURL":    urlData.OriginalURL,
			"updatedAt": urlData.UpdatedAt,
		}).Info("Updated shortURL")
		return nil
	}
}

// Delete removes a short URL and its corresponding original URL from the storage.
func (s *InMemoryStorage) Delete(ctx context.Context, shortURL string) error {
	select {
	case <-ctx.Done():
		s.logger.WithField("shortURL", shortURL).Warn("Delete operation cancelled")
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		if _, exists := s.urls[shortURL]; !exists {
			s.logger.WithField("shortURL", shortURL).Warn("Attempt to delete non-existent shortURL")
			return ErrShortURLNotFound
		}

		delete(s.urls, shortURL)
		s.count--
		s.logger.WithField("shortURL", shortURL).Info("Deleted shortURL")
		return nil
	}
}
