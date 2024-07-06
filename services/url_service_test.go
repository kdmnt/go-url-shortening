package services

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go-url-shortening/storage"
	"go-url-shortening/storage/mocks"
	"go-url-shortening/types"
	"sync"
	"testing"
)

func TestCreateShortURL(t *testing.T) {
	mockStorage := new(mocks.MockStorage)
	service := NewURLService(mockStorage)

	ctx := context.Background()
	originalURL := "https://example.com"

	t.Run("Success", func(t *testing.T) {
		mockStorage.On("GetShortURL", ctx, originalURL).Return("", storage.ErrShortURLNotFound).Once()
		mockStorage.On("Create", ctx, mock.AnythingOfType("types.URLData")).Return(nil).Once()

		urlData, err := service.CreateShortURL(ctx, originalURL)

		assert.NoError(t, err)
		assert.NotEmpty(t, urlData.ShortURL)
		assert.Equal(t, originalURL, urlData.OriginalURL)
		assert.False(t, urlData.CreatedAt.IsZero())
		assert.False(t, urlData.UpdatedAt.IsZero())
		mockStorage.AssertExpectations(t)
	})

	t.Run("ShortURLExists", func(t *testing.T) {
		existingShortURL := "abc123"

		mockStorage.On("GetShortURL", ctx, originalURL).Return(existingShortURL, storage.ErrShortURLExists).Once()

		_, err := service.CreateShortURL(ctx, originalURL)

		assert.Equal(t, ErrShortURLExists, err)
		mockStorage.AssertExpectations(t)
	})

	t.Run("StorageCapacityReached", func(t *testing.T) {
		mockStorage.On("GetShortURL", ctx, originalURL).Return("", storage.ErrShortURLNotFound).Once()
		mockStorage.On("Create", ctx, mock.AnythingOfType("types.URLData")).Return(storage.ErrStorageCapacityReached).Once()

		_, err := service.CreateShortURL(ctx, originalURL)

		assert.Equal(t, ErrStorageCapacityReached, err)
		mockStorage.AssertExpectations(t)
	})
}

func TestGetURLData(t *testing.T) {
	mockStorage := new(mocks.MockStorage)
	service := NewURLService(mockStorage)

	ctx := context.Background()
	shortURL := "abc123"
	originalURL := "https://example.com"

	t.Run("Success", func(t *testing.T) {
		mockStorage.On("GetURLData", ctx, shortURL).Return(types.URLData{OriginalURL: originalURL}, nil).Once()

		result, err := service.GetURLData(ctx, shortURL)

		assert.NoError(t, err)
		assert.Equal(t, originalURL, result.OriginalURL)
		mockStorage.AssertExpectations(t)
	})

	t.Run("ShortURLNotFound", func(t *testing.T) {
		mockStorage.On("GetURLData", ctx, shortURL).Return(types.URLData{}, storage.ErrShortURLNotFound).Once()

		_, err := service.GetURLData(ctx, shortURL)

		assert.Equal(t, ErrShortURLNotFound, err)
		mockStorage.AssertExpectations(t)
	})
}

func TestUpdateURL(t *testing.T) {
	mockStorage := new(mocks.MockStorage)
	service := NewURLService(mockStorage)

	ctx := context.Background()
	shortURL := "abc123"
	newURL := "https://newexample.com"

	t.Run("Success", func(t *testing.T) {
		mockStorage.On("GetURLData", ctx, shortURL).Return(types.URLData{OriginalURL: "https://oldexample.com"}, nil).Once()
		mockStorage.On("Update", ctx, mock.AnythingOfType("types.URLData")).Return(nil).Once()

		err := service.UpdateURL(ctx, shortURL, newURL)

		assert.NoError(t, err)
		mockStorage.AssertExpectations(t)
	})

	t.Run("ShortURLNotFound", func(t *testing.T) {
		mockStorage.On("GetURLData", ctx, shortURL).Return(types.URLData{}, storage.ErrShortURLNotFound).Once()

		err := service.UpdateURL(ctx, shortURL, newURL)

		assert.Equal(t, ErrShortURLNotFound, err)
		mockStorage.AssertExpectations(t)
	})
}

func TestDeleteURL(t *testing.T) {
	mockStorage := new(mocks.MockStorage)
	service := NewURLService(mockStorage)

	ctx := context.Background()
	shortURL := "abc123"

	t.Run("Success", func(t *testing.T) {
		mockStorage.On("Delete", ctx, shortURL).Return(nil).Once()

		err := service.DeleteURL(ctx, shortURL)

		assert.NoError(t, err)
		mockStorage.AssertExpectations(t)
	})

	t.Run("ShortURLNotFound", func(t *testing.T) {
		mockStorage.On("Delete", ctx, shortURL).Return(storage.ErrShortURLNotFound).Once()

		err := service.DeleteURL(ctx, shortURL)

		assert.Equal(t, ErrShortURLNotFound, err)
		mockStorage.AssertExpectations(t)
	})
}

func TestConcurrentAccess(t *testing.T) {
	mockStorage := new(mocks.MockStorage)
	service := NewURLService(mockStorage)

	ctx := context.Background()
	originalURL := "https://example.com"

	mockStorage.On("GetShortURL", ctx, originalURL).Return("", storage.ErrShortURLNotFound)
	mockStorage.On("Create", ctx, mock.AnythingOfType("types.URLData")).Return(nil)

	var wg sync.WaitGroup
	concurrentRequests := 100

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.CreateShortURL(ctx, originalURL)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
	mockStorage.AssertNumberOfCalls(t, "GetShortURL", concurrentRequests)
	mockStorage.AssertNumberOfCalls(t, "Create", concurrentRequests)
}
