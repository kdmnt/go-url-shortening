package handlers

import (
	"context"
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
	"testing"
	"time"
)

func TestRedirectURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		RateLimit:      10,
		RatePeriod:     time.Second,
		RequestTimeout: 5 * time.Second,
		ServerPort:     ":3000",
	}

	mockLogger := zap.NewNop()
	ctx := context.Background()

	tests := []struct {
		name           string
		shortURL       string
		mockGetURLData func(ctx context.Context, shortURL string) (types.URLData, error)
		mockLimiter    *rate.Limiter
		expectedStatus int
		expectedURL    string
		expectedBody   string
	}{
		{
			name:     "Valid short URL",
			shortURL: "abc123",
			mockGetURLData: func(ctx context.Context, shortURL string) (types.URLData, error) {
				return types.URLData{OriginalURL: "https://example.com"}, nil
			},
			mockLimiter:    rate.NewLimiter(rate.Every(time.Second), 10),
			expectedStatus: http.StatusMovedPermanently,
			expectedURL:    "https://example.com",
		},
		{
			name:     "Short URL not found",
			shortURL: "notfound",
			mockGetURLData: func(ctx context.Context, shortURL string) (types.URLData, error) {
				return types.URLData{OriginalURL: ""}, services.ErrShortURLNotFound
			},
			mockLimiter:    rate.NewLimiter(rate.Every(time.Second), 10),
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Short URL not found"}`,
		},
		{
			name:     "Service error",
			shortURL: "error",
			mockGetURLData: func(ctx context.Context, shortURL string) (types.URLData, error) {
				return types.URLData{OriginalURL: ""}, errors.New("service error")
			},
			mockLimiter:    rate.NewLimiter(rate.Every(time.Second), 10),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error":"Error retrieving URL"}`,
		},
		{
			name:     "Invalid original URL",
			shortURL: "invalid",
			mockGetURLData: func(ctx context.Context, shortURL string) (types.URLData, error) {
				return types.URLData{OriginalURL: "not-a-valid-url"}, nil
			},
			mockLimiter:    rate.NewLimiter(rate.Every(time.Second), 10),
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid redirect URL"}`,
		},
		{
			name:     "Request timeout",
			shortURL: "timeout",
			mockGetURLData: func(ctx context.Context, shortURL string) (types.URLData, error) {
				return types.URLData{OriginalURL: ""}, context.DeadlineExceeded
			},
			mockLimiter:    rate.NewLimiter(rate.Every(time.Second), 10),
			expectedStatus: http.StatusRequestTimeout,
			expectedBody:   `{"error":"Request timed out"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockURLService)
			mockService.On("GetURLData", mock.Anything, tt.shortURL).Return(tt.mockGetURLData(ctx, tt.shortURL))

			handler, err := NewURLHandler(ctx, mockService, cfg, mockLogger, tt.mockLimiter)
			require.NoError(t, err)

			router := gin.New()
			router.GET("/:short_url", handler.RedirectURL)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/"+tt.shortURL, nil)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus == http.StatusMovedPermanently {
				assert.Equal(t, tt.expectedURL, resp.Header().Get("Location"))
			} else {
				assert.JSONEq(t, tt.expectedBody, resp.Body.String())
			}
		})
	}
}
