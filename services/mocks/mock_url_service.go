package mocks

import (
	"context"
	"go-url-shortening/types"

	"github.com/stretchr/testify/mock"
)

// MockURLService is a mock URLService interface
type MockURLService struct {
	mock.Mock
}

func (m *MockURLService) CreateShortURL(ctx context.Context, originalURL string) (types.URLData, error) {
	args := m.Called(ctx, originalURL)
	return args.Get(0).(types.URLData), args.Error(1)
}

func (m *MockURLService) GetURLData(ctx context.Context, shortURL string) (types.URLData, error) {
	args := m.Called(ctx, shortURL)
	return args.Get(0).(types.URLData), args.Error(1)
}

func (m *MockURLService) UpdateURL(ctx context.Context, shortURL, newURL string) error {
	args := m.Called(ctx, shortURL, newURL)
	return args.Error(0)
}

func (m *MockURLService) DeleteURL(ctx context.Context, shortURL string) error {
	args := m.Called(ctx, shortURL)
	return args.Error(0)
}
