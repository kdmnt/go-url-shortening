package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-url-shortening/config"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("CORS headers are set", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, "*", resp.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "POST, GET, OPTIONS, PUT, DELETE", resp.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization", resp.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "nosniff", resp.Header().Get("X-Content-Type-Options"))
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		RateLimit:  10,
		RatePeriod: time.Second,
	}
	handler := &URLHandler{
		config: cfg,
	}

	router := gin.New()
	router.Use(handler.RateLimitMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	t.Run("Within rate limit", func(t *testing.T) {
		for i := 0; i < cfg.RateLimit; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = "192.0.2.1:1234" // Set a consistent IP address
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			assert.Equal(t, http.StatusOK, resp.Code)
			time.Sleep(time.Millisecond) // Small delay to ensure rate limiter updates
		}
	})

	t.Run("Exceeds rate limit", func(t *testing.T) {
		// Make one more request to exceed the limit
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "192.0.2.1:1234" // Use the same IP as before
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusTooManyRequests, resp.Code)
		assert.Contains(t, resp.Body.String(), "Rate limit exceeded")
	})

	t.Run("Rate limit resets after period", func(t *testing.T) {
		// Use a different IP to avoid interference from previous tests
		ip := "192.0.2.2:1234"

		// First, reach the rate limit
		for i := 0; i < cfg.RateLimit; i++ {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = ip
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, http.StatusOK, resp.Code)
		}

		// Verify that the next request is rate limited
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusTooManyRequests, resp.Code)

		// Wait for the rate limit period to pass
		time.Sleep(cfg.RatePeriod)

		// Verify that we can make a request again
		req = httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip // making sure we are using the same ip
		resp = httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code)
	})
}
