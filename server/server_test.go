package server

import (
	"context"
	"github.com/stretchr/testify/mock"
	"go-url-shortening/handlers"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go-url-shortening/config"
	"go-url-shortening/handlers/mocks"
	"go-url-shortening/storage"
	"go.uber.org/zap"
)

var setupURLHandlerFunc func(ctx context.Context, cfg *config.Config, store storage.Storage, logger *zap.Logger) (handlers.URLHandlerInterface, error)

func init() {
	setupURLHandlerFunc = setupURLHandler
}

func TestRun(t *testing.T) {
	logger := zap.NewNop()

	cfg := config.DefaultConfig()
	cfg.ServerPort = 3001 // Use a different port to avoid conflicts

	// Create a mock URL handler
	mockHandler := &mocks.MockURLHandler{}
	mockHandler.On("HealthCheck", mock.Anything).Run(func(args mock.Arguments) {
		c := args.Get(0).(*gin.Context)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	mockHandler.On("RateLimitMiddleware").Return(gin.HandlerFunc(func(c *gin.Context) {}))

	// Replace setupURLHandlerFunc with a test function
	originalSetupURLHandlerFunc := setupURLHandlerFunc
	setupURLHandlerFunc = func(ctx context.Context, cfg *config.Config, store storage.Storage, logger *zap.Logger) (handlers.URLHandlerInterface, error) {
		return mockHandler, nil
	}
	defer func() { setupURLHandlerFunc = originalSetupURLHandlerFunc }()

	// Create a test context that we can cancel
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run the server in a goroutine
	go func() {
		err := Run(logger, cfg)
		assert.NoError(t, err)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Make a request to the health check endpoint
	resp, err := http.Get("http://localhost:" + strconv.Itoa(cfg.ServerPort) + "/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Cancel the context to stop the server
	cancel()

	// Give the server a moment to shut down
	time.Sleep(100 * time.Millisecond)
}

func TestRunServerStartupFailure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := config.DefaultConfig()
	cfg.ServerPort = -1 // Invalid port to force startup failure

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Run the server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- Run(logger, cfg)
	}()

	// Wait for either the error or the timeout
	select {
	case err := <-errChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "listen tcp: address -1: invalid port")
	case <-ctx.Done():
		t.Fatal("Test timed out")
	}
}

func TestSetupURLHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := zap.NewNop()
	store := storage.NewInMemoryStorage(1000000, logger)

	ctx := context.Background()
	handler, err := setupURLHandler(ctx, cfg, store, logger)

	assert.NoError(t, err)
	assert.NotNil(t, handler)
}

func TestSetupRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := zap.NewNop()
	store := storage.NewInMemoryStorage(1000000, logger)

	ctx := context.Background()
	handler, err := setupURLHandler(ctx, cfg, store, logger)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	_, router := gin.CreateTestContext(w)

	router = setupRouter(handler, cfg)

	assert.NotNil(t, router)

	// Check if the expected routes are registered
	routes := router.Routes()
	expectedPaths := []string{
		"/api/v1/short",
		"/api/v1/short/:short_url",
		"/health",
		"/:short_url",
	}

	for _, path := range expectedPaths {
		found := false
		for _, route := range routes {
			if route.Path == path {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected route %s not found", path)
	}
}

func TestSetupServer(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ServerPort = 3002 // Use a different port to avoid conflicts
	w := httptest.NewRecorder()
	_, router := gin.CreateTestContext(w)

	server := setupServer(cfg, router)

	assert.NotNil(t, server)
	assert.Equal(t, ":"+strconv.Itoa(cfg.ServerPort), server.Addr)
	assert.Equal(t, router, server.Handler)
}

func TestStartServer(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ServerPort = 3003 // Use a different port to avoid conflicts
	w := httptest.NewRecorder()
	_, router := gin.CreateTestContext(w)
	server := setupServer(cfg, router)
	logger := zap.NewNop()

	// Start the server in a goroutine
	go startServer(server, logger)

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Try to connect to the server
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code) // Expect 404 as we haven't set up any routes

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := server.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestWaitForShutdown(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := context.Background()
	logger := zap.NewNop()
	mockHandler := &mocks.MockURLHandler{}
	mockHandler.On("HealthCheck", mock.Anything).Run(func(args mock.Arguments) {
		c := args.Get(0).(*gin.Context)
		c.JSON(http.StatusOK, gin.H{})
	}).Return()

	mockHandler.On("RateLimitMiddleware").Return(gin.HandlerFunc(func(c *gin.Context) {}))

	router := setupRouter(mockHandler, cfg)
	server := setupServer(cfg, router)

	// Start the server in a goroutine
	go startServer(server, logger)

	// Simulate SIGINT
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		err := p.Signal(os.Interrupt)
		if err != nil {
			return
		}
	}()

	// Run waitForShutdown in a goroutine
	done := make(chan error)
	go func() {
		done <- waitForShutdown(ctx, server, logger)
	}()

	// Wait for waitForShutdown to finish or timeout
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("waitForShutdown did not finish within the expected time")
	}
}
