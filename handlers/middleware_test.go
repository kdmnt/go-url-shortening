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

const (
	testIP = "192.0.2.1:1234"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CORS headers are set correctly for GET request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		CORSMiddleware()(c)

		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "POST, GET, OPTIONS, PUT, DELETE", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	})

	t.Run("OPTIONS request returns OK status", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("OPTIONS", "/", nil)
		CORSMiddleware()(c)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	cfg := &config.Config{
		RateLimit:  10,
		RatePeriod: time.Second,
	}
	handler := &URLHandler{
		config: cfg,
	}

	middleware := handler.RateLimitMiddleware()

	t.Run("Within rate limit", func(t *testing.T) {
		for i := 0; i < cfg.RateLimit; i++ {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Request.RemoteAddr = testIP

			middleware(c)

			assert.Equal(t, http.StatusOK, w.Code)
			// Small delay to ensure rate limiter updates
			time.Sleep(time.Millisecond)
		}
	})

	t.Run("Exceeds rate limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/", nil)
		c.Request.RemoteAddr = testIP

		middleware(c)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "Rate limit exceeded")
	})

	t.Run("Rate limit resets after period", func(t *testing.T) {
		ip := "192.0.2.2:1234"

		t.Run("Reach rate limit", func(t *testing.T) {
			for i := 0; i < cfg.RateLimit; i++ {
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = httptest.NewRequest("GET", "/", nil)
				c.Request.RemoteAddr = ip
				middleware(c)
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})

		t.Run("Verify rate limit", func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Request.RemoteAddr = ip
			middleware(c)
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		})

		t.Run("Wait for rate limit reset", func(t *testing.T) {
			time.Sleep(cfg.RatePeriod)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Request.RemoteAddr = ip
			middleware(c)
			assert.Equal(t, http.StatusOK, w.Code)
		})
	})
}
