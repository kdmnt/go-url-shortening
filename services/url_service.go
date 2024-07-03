package services

import (
	"context"
	"errors"
	"go-url-shortening/storage"
	"go-url-shortening/types"
	"go-url-shortening/utils"
	"time"
)

func handleStorageError(err error) error {
	switch {
	case errors.Is(err, storage.ErrShortURLExists):
		return ErrShortURLExists
	case errors.Is(err, storage.ErrStorageCapacityReached):
		return ErrStorageCapacityReached
	case errors.Is(err, storage.ErrShortURLNotFound):
		return ErrShortURLNotFound
	default:
		return err
	}
}

var (
	ErrShortURLExists         = errors.New("short URL already exists")
	ErrStorageCapacityReached = errors.New("storage capacity reached")
	ErrShortURLNotFound       = errors.New("short URL not found")
)

type URLService interface {
	CreateShortURL(ctx context.Context, originalURL string) (types.URLData, error)
	GetURLData(ctx context.Context, shortURL string) (types.URLData, error)
	UpdateURL(ctx context.Context, shortURL, newURL string) error
	DeleteURL(ctx context.Context, shortURL string) error
}

type urlService struct {
	store storage.Storage
}

func NewURLService(store storage.Storage) URLService {
	return &urlService{store: store}
}

func (s *urlService) CreateShortURL(ctx context.Context, originalURL string) (types.URLData, error) {
	// Check if the original URL already exists
	existingShortURL, err := s.store.GetShortURL(ctx, originalURL)
	if err == nil {
		// URL already exists, retrieve its data
		urlData, err := s.store.GetURLData(ctx, existingShortURL)
		if err != nil {
			return types.URLData{}, handleStorageError(err)
		}
		return urlData, nil
	}
	if !errors.Is(err, storage.ErrShortURLNotFound) {
		return types.URLData{}, handleStorageError(err)
	}

	// Generate new short URL
	shortURL, err := utils.GenerateShortURL()
	if err != nil {
		return types.URLData{}, err
	}

	now := time.Now()
	urlData := types.URLData{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = s.store.Create(ctx, urlData)
	if err != nil {
		return types.URLData{}, handleStorageError(err)
	}

	return urlData, nil
}

func (s *urlService) GetURLData(ctx context.Context, shortURL string) (types.URLData, error) {
	urlData, err := s.store.GetURLData(ctx, shortURL)
	if err != nil {
		return types.URLData{}, handleStorageError(err)
	}
	return urlData, nil
}

func (s *urlService) UpdateURL(ctx context.Context, shortURL, newURL string) error {
	urlData, err := s.store.GetURLData(ctx, shortURL)
	if err != nil {
		return handleStorageError(err)
	}

	urlData.OriginalURL = newURL
	urlData.UpdatedAt = time.Now()
	err = s.store.Update(ctx, urlData)
	if err != nil {
		return handleStorageError(err)
	}
	return nil
}

func (s *urlService) DeleteURL(ctx context.Context, shortURL string) error {
	err := s.store.Delete(ctx, shortURL)
	if err != nil {
		return handleStorageError(err)
	}
	return nil
}
