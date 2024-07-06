package storage

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-url-shortening/types"
	"go.uber.org/zap"
	"sync"
	"testing"
)

func TestInMemoryStorage(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	storage := NewInMemoryStorage(10, logger)

	t.Run("NewInMemoryStorage", func(t *testing.T) {
		// Test with capacity <= 0
		logger := zap.NewNop()
		storage := NewInMemoryStorage(0, logger)
		assert.Equal(t, 1000, storage.capacity, "Capacity should be set to default 1000 when input is 0")

		storage = NewInMemoryStorage(-5, logger)
		assert.Equal(t, 1000, storage.capacity, "Capacity should be set to default 1000 when input is negative")

		// Test with nil logger
		storage = NewInMemoryStorage(10, nil)
		assert.NotNil(t, storage.logger, "Logger should be initialized when input is nil")
		assert.IsType(t, (*zap.Logger)(nil), storage.logger, "Logger should be of type *zap.Logger")
	})

	t.Run("Create", func(t *testing.T) {
		err := storage.Create(ctx, types.URLData{ShortURL: "abc123", OriginalURL: "https://example.com"})
		assert.NoError(t, err)

		// Test duplicate creation
		err = storage.Create(ctx, types.URLData{ShortURL: "abc123", OriginalURL: "https://example.com"})
		assert.Equal(t, ErrShortURLExists, err)

		// Test capacity limit
		for i := 0; i < 9; i++ {
			err = storage.Create(ctx, types.URLData{ShortURL: fmt.Sprintf("test%d", i), OriginalURL: "https://test.com"})
			require.NoError(t, err)
		}
		err = storage.Create(ctx, types.URLData{ShortURL: "overflow", OriginalURL: "https://overflow.com"})
		assert.Equal(t, ErrStorageCapacityReached, err)

		// Test context cancellation
		logger := zap.NewNop()
		cancelStorage := NewInMemoryStorage(10, logger)
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err = cancelStorage.Create(cancelCtx, types.URLData{ShortURL: "cancelled", OriginalURL: "https://cancelled.com"})
		assert.Equal(t, context.Canceled, err, "Expected error to be context.Canceled")

		// Verify that the entry was not created
		_, err = cancelStorage.GetURLData(context.Background(), "cancelled")
		assert.Equal(t, ErrShortURLNotFound, err, "ShortURL should not have been added to the storage")

		// Verify that the count hasn't increased
		assert.Equal(t, 0, cancelStorage.count, "Storage count should remain 0")
	})

	t.Run("Read", func(t *testing.T) {
		// Test reading existing shortURL
		urlData, err := storage.GetURLData(ctx, "abc123")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", urlData.OriginalURL)

		// Test non-existent URL
		_, err = storage.GetURLData(ctx, "nonexistent")
		assert.Equal(t, ErrShortURLNotFound, err)

		// Test context cancellation
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = storage.GetURLData(cancelCtx, "abc123")
		assert.Equal(t, context.Canceled, err, "Expected error to be context.Canceled")
	})

	t.Run("Update", func(t *testing.T) {
		// Test updating existent URL
		storage.urls["abc123"] = types.URLData{ShortURL: "abc123", OriginalURL: "http://example.com"}
		err := storage.Update(ctx, types.URLData{ShortURL: "abc123", OriginalURL: "https://updated.com"})
		assert.NoError(t, err)

		urlData, err := storage.GetURLData(ctx, "abc123")
		assert.NoError(t, err)
		assert.Equal(t, "https://updated.com", urlData.OriginalURL)

		// Test updating non-existent URL
		err = storage.Update(ctx, types.URLData{ShortURL: "nonexistent", OriginalURL: "https://new.com"})
		assert.Equal(t, ErrShortURLNotFound, err)

		// Test context cancellation
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()
		err = storage.Update(cancelCtx, types.URLData{ShortURL: "abc123", OriginalURL: "https://cancelled.com"})
		assert.Equal(t, context.Canceled, err)

		// Verify that the URL was not updated after cancellation
		urlData, err = storage.GetURLData(ctx, "abc123")
		assert.NoError(t, err)
		assert.Equal(t, "https://updated.com", urlData.OriginalURL)
	})

	t.Run("Delete", func(t *testing.T) {
		// Test deleting existent URL
		storage.urls["abc123"] = types.URLData{OriginalURL: "http://example.com"}
		err := storage.Delete(ctx, "abc123")
		assert.NoError(t, err)

		_, err = storage.GetURLData(ctx, "abc123")
		assert.Equal(t, ErrShortURLNotFound, err)

		// Test deleting non-existent URL
		err = storage.Delete(ctx, "nonexistent")
		assert.Equal(t, ErrShortURLNotFound, err)

		// Test context cancellation
		logger := zap.NewNop()
		cancelStorage := NewInMemoryStorage(10, logger)
		shortURL := "shortURL"
		originalURL := "https://example.com"

		// Create an entry to delete
		err = cancelStorage.Create(context.Background(), types.URLData{ShortURL: shortURL, OriginalURL: originalURL})
		require.NoError(t, err)

		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()

		// Try to delete the entry with a cancelled context
		err = cancelStorage.Delete(cancelCtx, shortURL)
		assert.Equal(t, context.Canceled, err, "Expected error to be context.Canceled")

		// Verify that the entry was not deleted
		_, err = cancelStorage.GetURLData(context.Background(), shortURL)
		assert.NoError(t, err, "ShortURL should still exist in the storage")

		// Verify that the count hasn't decreased
		assert.Equal(t, 1, cancelStorage.count, "Storage count should remain 1")
	})

	t.Run("Concurrent operations", func(t *testing.T) {
		logger := zap.NewNop()
		storage := NewInMemoryStorage(1000000, logger)
		var wg sync.WaitGroup
		numOperations := 100

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				shortURL := fmt.Sprintf("short%d", i)
				originalURL := fmt.Sprintf("https://example.com/%d", i)

				err := storage.Create(context.Background(), types.URLData{ShortURL: shortURL, OriginalURL: originalURL})
				assert.NoError(t, err)

				urlData, err := storage.GetURLData(context.Background(), shortURL)
				assert.NoError(t, err)
				assert.Equal(t, originalURL, urlData.OriginalURL)

				newURL := fmt.Sprintf("https://updated.com/%d", i)
				err = storage.Update(context.Background(), types.URLData{ShortURL: shortURL, OriginalURL: newURL})
				assert.NoError(t, err)

				err = storage.Delete(context.Background(), shortURL)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		assert.Equal(t, 0, storage.count, "All entries should have been deleted")
	})

	t.Run("GetShortURL", func(t *testing.T) {
		logger := zap.NewNop()
		storage := NewInMemoryStorage(10, logger)

		// Create a URL
		originalURL := "https://example.com"
		shortURL := "abc123"
		err := storage.Create(ctx, types.URLData{ShortURL: shortURL, OriginalURL: originalURL})
		require.NoError(t, err)

		// Test getting existing short URL
		gotShortURL, err := storage.GetShortURL(ctx, originalURL)
		assert.NoError(t, err)
		assert.Equal(t, shortURL, gotShortURL)

		// Test getting non-existent URL
		_, err = storage.GetShortURL(ctx, "https://nonexistent.com")
		assert.Equal(t, ErrShortURLNotFound, err)

		// Test context cancellation
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = storage.GetShortURL(cancelCtx, originalURL)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("Storage count accuracy", func(t *testing.T) {
		logger := zap.NewNop()
		storage := NewInMemoryStorage(10, logger)

		// Create entries
		for i := 0; i < 5; i++ {
			err := storage.Create(ctx, types.URLData{ShortURL: fmt.Sprintf("short%d", i), OriginalURL: fmt.Sprintf("https://example%d.com", i)})
			require.NoError(t, err)
		}
		assert.Equal(t, 5, storage.count)

		// Update an entry (shouldn't change count)
		err := storage.Update(ctx, types.URLData{ShortURL: "short0", OriginalURL: "https://updated.com"})
		require.NoError(t, err)
		assert.Equal(t, 5, storage.count)

		// Delete an entry
		err = storage.Delete(ctx, "short1")
		require.NoError(t, err)
		assert.Equal(t, 4, storage.count)

		// Try to create a duplicate (shouldn't change count)
		err = storage.Create(ctx, types.URLData{ShortURL: "short2", OriginalURL: "https://duplicate.com"})
		assert.Equal(t, ErrShortURLExists, err)
		assert.Equal(t, 4, storage.count)
	})

	t.Run("Concurrent operations with specific scenarios", func(t *testing.T) {
		logger := zap.NewNop()
		storage := NewInMemoryStorage(1000, logger)
		var wg sync.WaitGroup
		numOperations := 100

		// Scenario 1: Concurrent creations of the same short URL
		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := storage.Create(context.Background(), types.URLData{ShortURL: "concurrent", OriginalURL: "https://example.com"})
				if err != nil && err != ErrShortURLExists {
					t.Errorf("Unexpected error: %v", err)
				}
			}()
		}
		wg.Wait()
		assert.Equal(t, 1, storage.count, "Only one entry should have been created")

		// Scenario 2: Concurrent reads and updates
		err := storage.Create(context.Background(), types.URLData{ShortURL: "readupdate", OriginalURL: "https://original.com"})
		require.NoError(t, err)

		for i := 0; i < numOperations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				if i%2 == 0 {
					_, err := storage.GetURLData(context.Background(), "readupdate")
					assert.NoError(t, err)
				} else {
					err := storage.Update(context.Background(), types.URLData{ShortURL: "readupdate", OriginalURL: fmt.Sprintf("https://updated%d.com", i)})
					assert.NoError(t, err)
				}
			}(i)
		}
		wg.Wait()
		assert.Equal(t, 2, storage.count, "Count should remain 2 after concurrent reads and updates")
	})
}
