package services

import (
	"context"
	"go-url-shortening/storage"
	"go-url-shortening/storage/mocks"
	"go-url-shortening/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		existingShortURL := "existingShortURL"
		existingURLData := types.URLData{
			ShortURL:    existingShortURL,
			OriginalURL: originalURL,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		mockStorage.On("GetShortURL", ctx, originalURL).Return(existingShortURL, nil).Once()
		mockStorage.On("GetURLData", ctx, existingShortURL).Return(existingURLData, nil).Once()

		urlData, err := service.CreateShortURL(ctx, originalURL)

		assert.NoError(t, err)
		assert.Equal(t, existingURLData, urlData)
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
