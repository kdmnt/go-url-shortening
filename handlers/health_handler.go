// Package handlers provides HTTP request handlers for the URL shortener service.
package handlers

import (
	"github.com/sirupsen/logrus"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck handles the health check endpoint.
// It returns a 200 OK status to indicate that the service is up and running.
func (h *URLHandler) HealthCheck(c *gin.Context) {
	h.logger.WithFields(logrus.Fields{
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	}).Info("Health check request")
	c.String(http.StatusOK, "OK")
}
