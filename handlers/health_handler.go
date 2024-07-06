// Package handlers provides HTTP request handlers for the URL shortener service.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthCheck handles the health check endpoint.
// It returns a 200 OK status to indicate that the service is up and running.
func (h *URLHandler) HealthCheck(c *gin.Context) {
	h.logger.Info("Health check request",
		zap.String("ip", c.ClientIP()),
		zap.String("user_agent", c.Request.UserAgent()),
	)
	c.String(http.StatusOK, "OK")
}
