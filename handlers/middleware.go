// Package handlers provides HTTP request handlers for the URL shortener service.
package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// CORSMiddleware adds CORS headers to the response.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Caveat make these configurable via Config ?
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware applies per-IP rate limiting to the given handler function.
// It checks if the request is within the rate limit before calling the next handler.
// If the rate limit is exceeded, it returns a 429 Too Many Requests error.
func (h *URLHandler) RateLimitMiddleware() gin.HandlerFunc {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(h.config.RateLimit), h.config.RateLimit)}
		}
		clients[ip].lastSeen = time.Now()
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next()
	}
}
