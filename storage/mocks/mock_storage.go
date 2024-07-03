package mocks

import (
	"context"
	"go-url-shortening/types"

	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock Storage interface
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Create(ctx context.Context, urlData types.URLData) error {
	args := m.Called(ctx, urlData)
	return args.Error(0)
}

func (m *MockStorage) GetURLData(ctx context.Context, shortURL string) (types.URLData, error) {
	args := m.Called(ctx, shortURL)
	return args.Get(0).(types.URLData), args.Error(1)
}

func (m *MockStorage) GetShortURL(ctx context.Context, originalURL string) (string, error) {
	args := m.Called(ctx, originalURL)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Update(ctx context.Context, urlData types.URLData) error {
	args := m.Called(ctx, urlData)
	return args.Error(0)
}

func (m *MockStorage) Delete(ctx context.Context, shortURL string) error {
	args := m.Called(ctx, shortURL)
	return args.Error(0)
}
