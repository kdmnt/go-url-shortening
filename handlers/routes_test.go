package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-url-shortening/config"
	"go-url-shortening/handlers/mocks"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTest() (*gin.Engine, *httptest.ResponseRecorder, *mocks.MockURLHandler, *config.Config) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, router := gin.CreateTestContext(w)
	mockHandler := new(mocks.MockURLHandler)
	cfg := config.DefaultConfig()
	return router, w, mockHandler, cfg
}

func TestRegisterRoutes(t *testing.T) {
	router, w, mockHandler, cfg := setupTest()

	// Mock RateLimitMiddleware for all subtests
	mockHandler.On("RateLimitMiddleware").Return(gin.HandlerFunc(func(c *gin.Context) {
		c.Next()
	}))
	RegisterRoutes(router, mockHandler, cfg)

	t.Run("Routes are registered correctly", func(t *testing.T) {
		routes := router.Routes()
		assert.Len(t, routes, 6)

		expectedRoutes := map[string][]string{
			"POST":    {"/api/v1/short"},
			"GET":     {"/api/v1/short/:short_url", "/health", "/:short_url"},
			"PUT":     {"/api/v1/short/:short_url"},
			"DELETE":  {"/api/v1/short/:short_url"},
			"OPTIONS": {"/api/v1/short"},
		}

		for _, route := range routes {
			expectedPaths, exists := expectedRoutes[route.Method]
			assert.True(t, exists, "Unexpected route method: %s", route.Method)
			assert.Contains(t, expectedPaths, route.Path, "Route path mismatch for method %s", route.Method)
		}
	})

	t.Run("CORS middleware is applied", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "/api/v1/short", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "POST, GET, OPTIONS, PUT, DELETE", w.Header().Get("Access-Control-Allow-Methods"))
	})

	t.Run("Rate limiting is applied and is enabled by default", func(t *testing.T) {
		mockHandler.AssertCalled(t, "RateLimitMiddleware")
	})

	t.Run("Rate limiting is not applied when disabled", func(t *testing.T) {
		newRouter, _, newMockHandler, newCfg := setupTest()
		newCfg.DisableRateLimit = true
		RegisterRoutes(newRouter, newMockHandler, newCfg)

		newMockHandler.AssertNotCalled(t, "RateLimitMiddleware")
	})
}
