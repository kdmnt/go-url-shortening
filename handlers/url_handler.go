// Package handlers provides HTTP request handlers for the URL shortener service.
package handlers

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"go-url-shortening/config"
	"go-url-shortening/services"
	"go-url-shortening/types"
	"golang.org/x/time/rate"
	"net/http"
)

const (
	invalidRequestBody  = "Invalid request body"
	errorCreatingURL    = "Error creating short URL"
	errorRetrievingURL  = "Error retrieving URL"
	errorUpdatingURL    = "Error updating URL"
	errorDeletingURL    = "Error deleting URL"
	errorTimeout        = "Request timed out"
	storageCapacityFull = "Storage capacity reached"
	shortURLExists      = "Short URL already exists"
	shortURLNotFound    = "Short URL not found"
	invalidURLProvided  = "Invalid URL provided"
)

// URLHandlerInterface defines the methods that a URL handler should implement.
type URLHandlerInterface interface {
	CreateShortURL(c *gin.Context)
	GetURLData(c *gin.Context)
	UpdateURL(c *gin.Context)
	DeleteURL(c *gin.Context)
	HealthCheck(c *gin.Context)
	RedirectURL(c *gin.Context)
	RateLimitMiddleware() gin.HandlerFunc
}

// handleError is a helper function to handle errors and send appropriate responses
func (h *URLHandler) handleError(c *gin.Context, err error, customMessages map[error]string) {
	var statusCode int
	var errorMessage string

	switch {
	case errors.Is(err, services.ErrShortURLExists):
		statusCode = http.StatusConflict
		errorMessage = customMessages[services.ErrShortURLExists]
	case errors.Is(err, services.ErrStorageCapacityReached):
		statusCode = http.StatusInsufficientStorage
		errorMessage = customMessages[services.ErrStorageCapacityReached]
	case errors.Is(err, services.ErrShortURLNotFound):
		statusCode = http.StatusNotFound
		errorMessage = customMessages[services.ErrShortURLNotFound]
	case errors.Is(err, context.DeadlineExceeded):
		statusCode = http.StatusRequestTimeout
		errorMessage = customMessages[context.DeadlineExceeded]
	default:
		h.logger.WithError(err).Error("Unexpected error")
		statusCode = http.StatusInternalServerError
		errorMessage = customMessages[err]
		if errorMessage == "" {
			errorMessage = "Internal server error"
		}
	}

	c.JSON(statusCode, gin.H{"error": errorMessage})
}

// URLHandler struct holds the dependencies for handling URL-related operations.
type URLHandler struct {
	service  services.URLService
	validate *validator.Validate
	limiter  *rate.Limiter
	config   *config.Config
	logger   *logrus.Logger
}

// NewURLHandler creates and returns a new URLHandler instance.
// It initializes the handler with the provided storage, a new validator,
// and a rate limiter configured with the settings from the config.
//
// Parameters:
//   - ctx: A context.Context for cancellation during initialization.
//   - store: An implementation of the storage.Storage interface for URL operations.
//   - cfg: A pointer to the Config struct containing application settings.
//
// Returns:
//   - A pointer to a new URLHandler instance and an error if initialization fails.
func NewURLHandler(ctx context.Context, service services.URLService, cfg *config.Config, logger *logrus.Logger, limiter *rate.Limiter) (URLHandlerInterface, error) {
	if service == nil {
		return nil, errors.New("service cannot be nil")
	}
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	if limiter == nil {
		return nil, errors.New("limiter cannot be nil")
	}
	if cfg.RateLimit <= 0 || cfg.RatePeriod <= 0 {
		return nil, errors.New("invalid rate limit configuration")
	}

	handler := &URLHandler{
		service:  service,
		validate: validator.New(),
		limiter:  rate.NewLimiter(rate.Every(cfg.RatePeriod), cfg.RateLimit),
		config:   cfg,
		logger:   logger,
	}

	// Perform any initialization that might be cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Initialization completed successfully
	}

	return handler, nil
}

// CreateShortURL handles the creation of a new shortened URL.
// It validates the input, generates a short URL, and stores it in the database.
func (h *URLHandler) CreateShortURL(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.config.RequestTimeout)
	defer cancel()

	var input types.URLRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.WithError(err).WithField("input", input).Error("Error decoding request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidRequestBody})
		return
	}

	// Validate the input
	if err := h.validate.Struct(input); err != nil {
		h.logger.WithError(err).WithField("input", input).Error("Invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": invalidURLProvided})
		return
	}

	urlData, err := h.service.CreateShortURL(ctx, input.URL)
	if err != nil {
		h.handleError(c, err, map[error]string{
			services.ErrShortURLExists:         shortURLExists,
			services.ErrStorageCapacityReached: storageCapacityFull,
			context.DeadlineExceeded:           errorTimeout,
			nil:                                errorCreatingURL,
		})
		return
	}

	response := types.URLResponse{
		ShortURL:    urlData.ShortURL,
		OriginalURL: urlData.OriginalURL,
		CreatedAt:   urlData.CreatedAt,
		UpdatedAt:   urlData.UpdatedAt,
	}
	c.JSON(http.StatusCreated, response)
}

// GetURLData retrieves the original URL for a given short URL.
// It returns the original URL in a JSON response if found, or an appropriate error if not found or if an error occurs.
func (h *URLHandler) GetURLData(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.config.RequestTimeout)
	defer cancel()

	shortURL := c.Param("short_url")

	urlData, err := h.service.GetURLData(ctx, shortURL)
	if err != nil {
		h.handleError(c, err, map[error]string{
			services.ErrShortURLNotFound: shortURLNotFound,
			context.DeadlineExceeded:     errorTimeout,
			nil:                          errorRetrievingURL,
		})
		return
	}

	response := types.URLResponse{
		ShortURL:    urlData.ShortURL,
		OriginalURL: urlData.OriginalURL,
		CreatedAt:   urlData.CreatedAt,
		UpdatedAt:   urlData.UpdatedAt,
	}
	c.JSON(http.StatusOK, response)
}

// UpdateURL updates the original URL for a given short URL.
// It validates the input, updates the URL in storage, and returns the updated URL pair in a JSON response.
// If the short URL is not found or an error occurs, it returns an appropriate error response.
func (h *URLHandler) UpdateURL(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.config.RequestTimeout)
	defer cancel()

	shortURL := c.Param("short_url")

	var input types.URLRequest

	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.WithError(err).WithField("input", input).Error("Error decoding request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.validate.Struct(input); err != nil {
		h.logger.WithError(err).WithField("input", input).Error("Invalid input")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL provided"})
		return
	}

	err := h.service.UpdateURL(ctx, shortURL, input.URL)
	if err != nil {
		h.handleError(c, err, map[error]string{
			services.ErrShortURLNotFound: shortURLNotFound,
			context.DeadlineExceeded:     errorTimeout,
			nil:                          errorUpdatingURL,
		})
		return
	}

	urlData, err := h.service.GetURLData(ctx, shortURL)
	if err != nil {
		h.handleError(c, err, map[error]string{
			services.ErrShortURLNotFound: shortURLNotFound,
			context.DeadlineExceeded:     errorTimeout,
			nil:                          errorRetrievingURL,
		})
		return
	}

	response := types.URLResponse{
		ShortURL:    urlData.ShortURL,
		OriginalURL: urlData.OriginalURL,
		CreatedAt:   urlData.CreatedAt,
		UpdatedAt:   urlData.UpdatedAt,
	}
	c.JSON(http.StatusOK, response)
}

// DeleteURL removes a short URL and its corresponding original URL from storage.
// It returns a 204 No Content status if successful, or an appropriate error response if the short URL is not found or an error occurs.
func (h *URLHandler) DeleteURL(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.config.RequestTimeout)
	defer cancel()

	shortURL := c.Param("short_url")

	err := h.service.DeleteURL(ctx, shortURL)
	if err != nil {
		h.handleError(c, err, map[error]string{
			services.ErrShortURLNotFound: shortURLNotFound,
			context.DeadlineExceeded:     errorTimeout,
			nil:                          errorDeletingURL,
		})
		return
	}

	c.Status(http.StatusNoContent)
}
