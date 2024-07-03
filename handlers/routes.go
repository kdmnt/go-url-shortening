// Package handlers provides HTTP request handlers for the URL shortener service.
package handlers

import (
	"github.com/gin-gonic/gin"
	"go-url-shortening/config"
)

// RegisterRoutes sets up all the routes for the URL shortener service.
// It registers all the API endpoints with their respective handlers,
// and applies middleware such as rate limiting and CORS.
func RegisterRoutes(r *gin.Engine, handler URLHandlerInterface, config *config.Config) {
	// Apply CORS middleware to all routes
	r.Use(CORSMiddleware())

	// API routes
	v1 := r.Group("/api/v1")
	if !config.DisableRateLimit {
		v1.Use(handler.RateLimitMiddleware())
	}
	{
		// Short URL routes
		short := v1.Group("/short")
		{
			short.POST("", handler.CreateShortURL)
			short.GET("/:short_url", handler.GetURLData)
			short.PUT("/:short_url", handler.UpdateURL)
			short.DELETE("/:short_url", handler.DeleteURL)
		}

		// Health check route
		if !config.DisableRateLimit {
			r.GET("/health", handler.RateLimitMiddleware(), handler.HealthCheck)
		} else {
			r.GET("/health", handler.HealthCheck)
		}
	}

	// Redirection route (not under /api/v1 as it's user-facing)
	if !config.DisableRateLimit {
		r.GET("/:short_url", handler.RateLimitMiddleware(), handler.RedirectURL)
	} else {
		r.GET("/:short_url", handler.RedirectURL)
	}
}
