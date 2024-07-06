package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go-url-shortening/config"
	"go-url-shortening/services"
	"go-url-shortening/services/mocks"
	"go-url-shortening/types"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewURLHandler(t *testing.T) {
	tests := []struct {
		name        string
		service     services.URLService
		cfg         *config.Config
		logger      *zap.Logger
		limiter     *rate.Limiter
		expectedErr string
	}{
		{
			name:    "Valid configuration",
			service: &mocks.MockURLService{},
			cfg: &config.Config{
				RateLimit:      10,
				RatePeriod:     time.Second,
				RequestTimeout: 5 * time.Second,
				ServerPort:     ":3000",
			},
			logger:      zap.NewNop(),
			limiter:     rate.NewLimiter(rate.Every(time.Second), 10),
			expectedErr: "",
		},
		{
			name:        "Nil service",
			service:     nil,
			cfg:         &config.Config{},
			logger:      zap.NewNop(),
			limiter:     rate.NewLimiter(rate.Every(time.Second), 10),
			expectedErr: "service cannot be nil",
		},
		{
			name:        "Nil config",
			service:     &mocks.MockURLService{},
			cfg:         nil,
			logger:      zap.NewNop(),
			limiter:     rate.NewLimiter(rate.Every(time.Second), 10),
			expectedErr: "config cannot be nil",
		},
		{
			name:        "Nil logger",
			service:     &mocks.MockURLService{},
			cfg:         &config.Config{},
			logger:      nil,
			limiter:     rate.NewLimiter(rate.Every(time.Second), 10),
			expectedErr: "logger cannot be nil",
		},
		{
			name:        "Nil limiter",
			service:     &mocks.MockURLService{},
			cfg:         &config.Config{},
			logger:      zap.NewNop(),
			limiter:     nil,
			expectedErr: "limiter cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewURLHandler(context.Background(), tt.service, tt.cfg, tt.logger, tt.limiter)

			if tt.expectedErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, handler)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, handler)

				// Type assertion to access the concrete type
				concreteHandler, ok := handler.(*URLHandler)
				require.True(t, ok, "Handler is not of type *URLHandler")

				assert.Equal(t, tt.service, concreteHandler.service)
				assert.Equal(t, tt.cfg, concreteHandler.config)
				assert.Equal(t, tt.logger, concreteHandler.logger)
				assert.Equal(t, tt.limiter, concreteHandler.limiter)
				assert.NotNil(t, concreteHandler.validate)
			}
		})
	}
}

func TestNewURLHandlerWithCancelledContext(t *testing.T) {
	service := &mocks.MockURLService{}
	cfg := &config.Config{
		RateLimit:      10,
		RatePeriod:     time.Second,
		RequestTimeout: 5 * time.Second,
		ServerPort:     ":3000",
	}
	logger := zap.NewNop()
	limiter := rate.NewLimiter(rate.Every(time.Second), 10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handler, err := NewURLHandler(ctx, service, cfg, logger, limiter)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Nil(t, handler)
}

func TestNewURLHandlerReturnsCorrectInterface(t *testing.T) {
	service := &mocks.MockURLService{}
	cfg := &config.Config{
		RateLimit:      10,
		RatePeriod:     time.Second,
		RequestTimeout: 5 * time.Second,
		ServerPort:     ":3000",
	}
	logger := zap.NewNop()
	limiter := rate.NewLimiter(rate.Every(time.Second), 10)

	handler, err := NewURLHandler(context.Background(), service, cfg, logger, limiter)

	require.NoError(t, err)
	assert.NotNil(t, handler)

	_, ok := handler.(URLHandlerInterface)
	assert.True(t, ok, "Handler does not implement URLHandlerInterface")
}

func setupTestHandler() (URLHandlerInterface, error) {
	cfg := &config.Config{
		RateLimit:      10,
		RatePeriod:     time.Second,
		RequestTimeout: 5 * time.Second,
		ServerPort:     ":3000",
	}
	mockService := &mocks.MockURLService{}
	logger := zap.NewNop()
	limiter := rate.NewLimiter(rate.Every(time.Second), 10)
	return NewURLHandler(context.Background(), mockService, cfg, logger, limiter)
}

func TestCreateShortURL(t *testing.T) {
	handler, err := setupTestHandler()
	require.NoError(t, err)

	tests := []struct {
		name               string
		inputURL           string
		expectedStatus     int
		mockCreateShortURL func(ctx context.Context, originalURL string) (types.URLData, error)
	}{
		{
			name:           "Context Deadline Exceeded",
			inputURL:       "https://example.com",
			expectedStatus: http.StatusRequestTimeout,
			mockCreateShortURL: func(ctx context.Context, originalURL string) (types.URLData, error) {
				return types.URLData{}, context.DeadlineExceeded
			},
		},
		{
			name:           "Valid URL",
			inputURL:       "https://example.com",
			expectedStatus: http.StatusCreated,
			mockCreateShortURL: func(ctx context.Context, originalURL string) (types.URLData, error) {
				return types.URLData{ShortURL: "mockedShortURL", OriginalURL: originalURL, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			},
		},
		{
			name:           "Valid URL with space",
			inputURL:       "https://example.com/search?q=with space",
			expectedStatus: http.StatusCreated,
			mockCreateShortURL: func(ctx context.Context, originalURL string) (types.URLData, error) {
				return types.URLData{ShortURL: "mockedShortURL", OriginalURL: originalURL, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			},
		},
		{
			name:           "Invalid URL",
			inputURL:       "not-a-url",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Very Long URL",
			inputURL:       "https://" + strings.Repeat(string('a'), 2000) + ".com",
			expectedStatus: http.StatusCreated,
			mockCreateShortURL: func(ctx context.Context, originalURL string) (types.URLData, error) {
				return types.URLData{ShortURL: "mockedShortURL", OriginalURL: originalURL, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			},
		},
		{
			name:           "Very Short URL",
			inputURL:       "http://a.co",
			expectedStatus: http.StatusCreated,
			mockCreateShortURL: func(ctx context.Context, originalURL string) (types.URLData, error) {
				return types.URLData{ShortURL: "mockedShortURL", OriginalURL: originalURL, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
			},
		},
		{
			name:           "Empty URL",
			inputURL:       "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service CreateShortURL fails with ErrShortURLExists",
			inputURL:       "https://example.com",
			expectedStatus: http.StatusConflict,
			mockCreateShortURL: func(ctx context.Context, originalURL string) (types.URLData, error) {
				return types.URLData{}, services.ErrShortURLExists
			},
		},
		{
			name:           "Service CreateShortURL fails with ErrStorageCapacityReached",
			inputURL:       "https://example.com",
			expectedStatus: http.StatusInsufficientStorage,
			mockCreateShortURL: func(ctx context.Context, originalURL string) (types.URLData, error) {
				return types.URLData{}, services.ErrStorageCapacityReached
			},
		},
		{
			name:           "Service CreateShortURL fails with unknown error",
			inputURL:       "https://example.com",
			expectedStatus: http.StatusInternalServerError,
			mockCreateShortURL: func(ctx context.Context, originalURL string) (types.URLData, error) {
				return types.URLData{}, errors.New("unknown error")
			},
		},
		{
			name:           "Invalid JSON input",
			inputURL:       "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock service
			mockService := &mocks.MockURLService{}

			// Set up mock service
			if tt.mockCreateShortURL != nil {
				mockService.On("CreateShortURL", mock.Anything, tt.inputURL).Return(tt.mockCreateShortURL(context.Background(), tt.inputURL))
			}

			urlHandler, ok := handler.(*URLHandler)
			require.True(t, ok)
			urlHandler.service = mockService

			var req *http.Request
			var rr *httptest.ResponseRecorder

			if tt.name == "Invalid JSON input" {
				req, _ = http.NewRequest("POST", "/api/v1/short", bytes.NewBufferString("invalid json"))
			} else {
				body, _ := json.Marshal(types.URLRequest{URL: tt.inputURL})
				req, _ = http.NewRequest("POST", "/api/v1/short", bytes.NewBuffer(body))
			}

			rr = httptest.NewRecorder()

			c, _ := gin.CreateTestContext(rr)
			c.Request = req
			handler.CreateShortURL(c)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response types.URLResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotEmpty(t, response.ShortURL)
				assert.Equal(t, tt.inputURL, response.OriginalURL)
			} else if tt.name == "Invalid JSON input" {
				var errorResponse map[string]string
				err := json.Unmarshal(rr.Body.Bytes(), &errorResponse)
				require.NoError(t, err)
				assert.Equal(t, "Invalid request body", errorResponse["error"])
			}
		})
	}
}

func TestGetURLData(t *testing.T) {
	handler, err := setupTestHandler()
	require.NoError(t, err)

	tests := []struct {
		name           string
		shortURL       string
		expectedStatus int
		expectedURL    string
		mockGetURLData func(ctx context.Context, shortURL string) (types.URLData, error)
	}{
		{
			name:           "Context Deadline Exceeded",
			shortURL:       "timeout",
			expectedStatus: http.StatusRequestTimeout,
			expectedURL:    "",
			mockGetURLData: func(ctx context.Context, shortURL string) (types.URLData, error) {
				return types.URLData{OriginalURL: ""}, context.DeadlineExceeded
			},
		},
		{
			name:           "Valid short URL",
			shortURL:       "abc123",
			expectedStatus: http.StatusOK,
			expectedURL:    "https://example.com",
			mockGetURLData: func(ctx context.Context, shortURL string) (types.URLData, error) {
				return types.URLData{
					ShortURL:    "abc123",
					OriginalURL: "https://example.com",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil
			},
		},
		{
			name:           "Short URL not found",
			shortURL:       "notfound",
			expectedStatus: http.StatusNotFound,
			expectedURL:    "",
			mockGetURLData: func(ctx context.Context, shortURL string) (types.URLData, error) {
				return types.URLData{OriginalURL: ""}, services.ErrShortURLNotFound
			},
		},
		{
			name:           "Service error",
			shortURL:       "error",
			expectedStatus: http.StatusInternalServerError,
			expectedURL:    "",
			mockGetURLData: func(ctx context.Context, shortURL string) (types.URLData, error) {
				return types.URLData{OriginalURL: ""}, errors.New("service error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock service
			mockService := &mocks.MockURLService{}

			// Set up mock service
			mockService.On("GetURLData", mock.Anything, tt.shortURL).Return(tt.mockGetURLData(context.Background(), tt.shortURL))

			urlHandler, ok := handler.(*URLHandler)
			require.True(t, ok)
			urlHandler.service = mockService

			// Create a new request
			req, _ := http.NewRequest("GET", "/api/v1/short/"+tt.shortURL, nil)
			rr := httptest.NewRecorder()

			// We need to set up the router to handle the request
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.GET("/api/v1/short/:short_url", handler.GetURLData)
			router.ServeHTTP(rr, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response types.URLResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.shortURL, response.ShortURL, "Short URL in response should match")
				assert.Equal(t, tt.expectedURL, response.OriginalURL, "Original URL in response should match")
				assert.NotZero(t, response.CreatedAt, "CreatedAt should not be zero")
				assert.NotZero(t, response.UpdatedAt, "UpdatedAt should not be zero")
			}
		})
	}
}

func TestUpdateURL(t *testing.T) {
	handler, err := setupTestHandler()
	require.NoError(t, err)

	tests := []struct {
		name           string
		shortURL       types.URLData
		inputURL       types.URLData
		expectedStatus int
		mockUpdateURL  func(ctx context.Context, shortURL, newURL string) error
	}{
		{
			name:           "Context Deadline Exceeded",
			shortURL:       types.URLData{ShortURL: "timeout"},
			inputURL:       types.URLData{OriginalURL: "https://newexample.com"},
			expectedStatus: http.StatusRequestTimeout,
			mockUpdateURL: func(ctx context.Context, shortURL, newURL string) error {
				return context.DeadlineExceeded
			},
		},
		{
			name:           "Valid Update",
			shortURL:       types.URLData{ShortURL: "abc123"},
			inputURL:       types.URLData{OriginalURL: "https://newexample.com"},
			expectedStatus: http.StatusOK,
			mockUpdateURL: func(ctx context.Context, shortURL, newURL string) error {
				return nil
			},
		},
		{
			name:           "Invalid URL",
			shortURL:       types.URLData{ShortURL: "abc123"},
			inputURL:       types.URLData{OriginalURL: "not-a-url"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Short URL Not Found",
			shortURL:       types.URLData{ShortURL: "notfound"},
			inputURL:       types.URLData{OriginalURL: "https://newexample.com"},
			expectedStatus: http.StatusNotFound,
			mockUpdateURL: func(ctx context.Context, shortURL, newURL string) error {
				return services.ErrShortURLNotFound
			},
		},
		{
			name:           "Service Error",
			shortURL:       types.URLData{ShortURL: "error"},
			inputURL:       types.URLData{OriginalURL: "https://newexample.com"},
			expectedStatus: http.StatusInternalServerError,
			mockUpdateURL: func(ctx context.Context, shortURL, newURL string) error {
				return errors.New("service error")
			},
		},
		{
			name:           "Valid URL with space",
			shortURL:       types.URLData{ShortURL: "abc123"},
			inputURL:       types.URLData{OriginalURL: "https://example.com/search?q=with space"},
			expectedStatus: http.StatusOK,
			mockUpdateURL: func(ctx context.Context, shortURL, newURL string) error {
				return nil
			},
		},
		{
			name:           "Very Long URL",
			shortURL:       types.URLData{ShortURL: "abc123"},
			inputURL:       types.URLData{OriginalURL: "https://" + strings.Repeat(string('a'), 2000) + ".com"},
			expectedStatus: http.StatusOK,
			mockUpdateURL: func(ctx context.Context, shortURL, newURL string) error {
				return nil
			},
		},
		{
			name:           "Very Short URL",
			shortURL:       types.URLData{ShortURL: "abc123"},
			inputURL:       types.URLData{OriginalURL: "http://a.co"},
			expectedStatus: http.StatusOK,
			mockUpdateURL: func(ctx context.Context, shortURL, newURL string) error {
				return nil
			},
		},
		{
			name:           "Empty URL",
			shortURL:       types.URLData{ShortURL: "abc123"},
			inputURL:       types.URLData{OriginalURL: ""},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Service UpdateURL fails with unknown error",
			shortURL:       types.URLData{ShortURL: "abc123"},
			inputURL:       types.URLData{OriginalURL: "https://example.com"},
			expectedStatus: http.StatusInternalServerError,
			mockUpdateURL: func(ctx context.Context, shortURL, newURL string) error {
				return errors.New("unknown error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock service
			mockService := &mocks.MockURLService{}

			// Set up mock service
			if tt.mockUpdateURL != nil {
				mockService.On("UpdateURL", mock.Anything, tt.shortURL.ShortURL, tt.inputURL.OriginalURL).Return(tt.mockUpdateURL(context.Background(), tt.shortURL.ShortURL, tt.inputURL.OriginalURL))
				mockService.On("GetURLData", mock.Anything, tt.shortURL.ShortURL).Return(types.URLData{
					ShortURL:    tt.shortURL.ShortURL,
					OriginalURL: tt.inputURL.OriginalURL,
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}, nil)
			}

			urlHandler, ok := handler.(*URLHandler)
			require.True(t, ok)
			urlHandler.service = mockService

			body, _ := json.Marshal(types.URLRequest{URL: tt.inputURL.OriginalURL})
			req, _ := http.NewRequest("PUT", "/api/v1/short/"+tt.shortURL.ShortURL, bytes.NewBuffer(body))
			rr := httptest.NewRecorder()

			// We need to set up the router to handle the request
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.PUT("/api/v1/short/:short_url", handler.UpdateURL)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var response types.URLResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.shortURL.ShortURL, response.ShortURL)
				assert.Equal(t, tt.inputURL.OriginalURL, response.OriginalURL)
				assert.NotZero(t, response.CreatedAt, "CreatedAt should not be zero")
				assert.NotZero(t, response.UpdatedAt, "UpdatedAt should not be zero")
			}
		})
	}
}

func TestDeleteURL(t *testing.T) {
	handler, err := setupTestHandler()
	require.NoError(t, err)

	tests := []struct {
		name           string
		shortURL       string
		expectedStatus int
		mockDeleteURL  func(ctx context.Context, shortURL string) error
	}{
		{
			name:           "Context Deadline Exceeded",
			shortURL:       "timeout",
			expectedStatus: http.StatusRequestTimeout,
			mockDeleteURL: func(ctx context.Context, shortURL string) error {
				return context.DeadlineExceeded
			},
		},
		{
			name:           "Valid Delete",
			shortURL:       "abc123",
			expectedStatus: http.StatusNoContent,
			mockDeleteURL: func(ctx context.Context, shortURL string) error {
				return nil
			},
		},
		{
			name:           "Short URL Not Found",
			shortURL:       "notfound",
			expectedStatus: http.StatusNotFound,
			mockDeleteURL: func(ctx context.Context, shortURL string) error {
				return services.ErrShortURLNotFound
			},
		},
		{
			name:           "Service Error",
			shortURL:       "error",
			expectedStatus: http.StatusInternalServerError,
			mockDeleteURL: func(ctx context.Context, shortURL string) error {
				return errors.New("service error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock service
			mockService := &mocks.MockURLService{}

			// Set up mock service
			mockService.On("DeleteURL", mock.Anything, tt.shortURL).Return(tt.mockDeleteURL(context.Background(), tt.shortURL))

			urlHandler, ok := handler.(*URLHandler)
			require.True(t, ok)
			urlHandler.service = mockService

			req, _ := http.NewRequest("DELETE", "/api/v1/short/"+tt.shortURL, nil)
			rr := httptest.NewRecorder()

			// We need to set up the router to handle the request
			gin.SetMode(gin.TestMode)
			router := gin.New()
			router.DELETE("/api/v1/short/:short_url", handler.DeleteURL)
			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
