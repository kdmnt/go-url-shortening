// Package handlers provides HTTP request handlers for the URL shortener service.
package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"go-url-shortening/services"
)

const (
	errShortURLNotFound   = "Short URL not found"
	errRequestTimeout     = "Request timed out"
	errRetrievingURL      = "Error retrieving URL"
	errInvalidRedirectURL = "Invalid redirect URL"
)

// RedirectURL handles the redirection from a short URL to its original URL.
// It retrieves the original URL associated with the given short URL from the storage
// and performs an HTTP redirect to that URL.
func (h *URLHandler) RedirectURL(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.config.RequestTimeout)
	defer cancel()

	shortURL := c.Param("short_url")

	urlData, err := h.service.GetURLData(ctx, shortURL)
	if err != nil {
		h.handleRedirectError(c, err, shortURL)
		return
	}

	// Validate the original URL to prevent open redirects
	if err := h.validate.Var(urlData.OriginalURL, "url"); err != nil {
		h.handleInvalidRedirectURL(c, shortURL, urlData.OriginalURL)
		return
	}

	h.logRedirect(c, shortURL, urlData.OriginalURL)
	c.Redirect(http.StatusMovedPermanently, urlData.OriginalURL)
}

func (h *URLHandler) handleRedirectError(c *gin.Context, err error, shortURL string) {
	switch {
	case errors.Is(err, services.ErrShortURLNotFound):
		h.logger.WithField("short_url", shortURL).Info("Short URL not found")
		c.JSON(http.StatusNotFound, gin.H{"error": errShortURLNotFound})
	case errors.Is(err, context.DeadlineExceeded):
		h.logger.WithField("short_url", shortURL).Warn("Request timed out")
		c.JSON(http.StatusRequestTimeout, gin.H{"error": errRequestTimeout})
	default:
		h.logger.WithFields(logrus.Fields{
			"short_url": shortURL,
			"error":     err,
		}).Error("Error retrieving URL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": errRetrievingURL})
	}
}

func (h *URLHandler) handleInvalidRedirectURL(c *gin.Context, shortURL, originalURL string) {
	h.logger.WithFields(logrus.Fields{
		"short_url":    shortURL,
		"original_url": originalURL,
	}).Warn("Invalid original URL")
	c.JSON(http.StatusBadRequest, gin.H{"error": errInvalidRedirectURL})
}

func (h *URLHandler) logRedirect(c *gin.Context, shortURL, originalURL string) {
	h.logger.WithFields(logrus.Fields{
		"short_url":    shortURL,
		"original_url": originalURL,
		"ip":           c.ClientIP(),
		"user_agent":   c.Request.UserAgent(),
	}).Info("Redirecting")
}
