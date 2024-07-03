package handlers

import (
	"github.com/stretchr/testify/mock"
	"go-url-shortening/config"
	"go-url-shortening/handlers/mocks"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTest() (*gin.Engine, *mocks.MockURLHandler, *config.Config) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	mockHandler := &mocks.MockURLHandler{}
	mockHandler.On("RateLimitMiddleware").Return(gin.HandlerFunc(func(c *gin.Context) {}))
	cfg := config.DefaultConfig()
	return router, mockHandler, cfg
}

func TestRegisterRoutes_CreateShortURL(t *testing.T) {
	router, mockHandler, cfg := setupTest()
	mockHandler.On("CreateShortURL", mock.Anything).Run(func(args mock.Arguments) {
		c := args.Get(0).(*gin.Context)
		c.JSON(http.StatusCreated, gin.H{})
	}).Return()

	RegisterRoutes(router, mockHandler, cfg)

	req, _ := http.NewRequest("POST", "/api/v1/short", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}
}

func TestRegisterRoutes_GetURLData(t *testing.T) {
	router, mockHandler, cfg := setupTest()
	mockHandler.On("GetURLData", mock.Anything).Run(func(args mock.Arguments) {
		c := args.Get(0).(*gin.Context)
		c.JSON(http.StatusOK, gin.H{})
	}).Return()

	RegisterRoutes(router, mockHandler, cfg)

	req, _ := http.NewRequest("GET", "/api/v1/short/abc123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestRegisterRoutes_UpdateURL(t *testing.T) {
	router, mockHandler, cfg := setupTest()
	mockHandler.On("UpdateURL", mock.Anything).Run(func(args mock.Arguments) {
		c := args.Get(0).(*gin.Context)
		c.JSON(http.StatusOK, gin.H{})
	}).Return()

	RegisterRoutes(router, mockHandler, cfg)

	req, _ := http.NewRequest("PUT", "/api/v1/short/abc123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestRegisterRoutes_DeleteURL(t *testing.T) {
	router, mockHandler, cfg := setupTest()
	mockHandler.On("DeleteURL", mock.Anything).Run(func(args mock.Arguments) {
		c := args.Get(0).(*gin.Context)
		c.JSON(http.StatusNoContent, gin.H{})
	}).Return()

	RegisterRoutes(router, mockHandler, cfg)

	req, _ := http.NewRequest("DELETE", "/api/v1/short/abc123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}

func TestRegisterRoutes_HealthCheck(t *testing.T) {
	router, mockHandler, cfg := setupTest()
	mockHandler.On("HealthCheck", mock.Anything).Run(func(args mock.Arguments) {
		c := args.Get(0).(*gin.Context)
		c.JSON(http.StatusOK, gin.H{})
	}).Return()

	RegisterRoutes(router, mockHandler, cfg)

	req, _ := http.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestRegisterRoutes_RedirectURL(t *testing.T) {
	router, mockHandler, cfg := setupTest()
	mockHandler.On("RedirectURL", mock.Anything).Run(func(args mock.Arguments) {
		c := args.Get(0).(*gin.Context)
		c.Redirect(http.StatusFound, "https://example.com")
	}).Return()

	RegisterRoutes(router, mockHandler, cfg)

	req, _ := http.NewRequest("GET", "/abc123", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if status := resp.Code; status != http.StatusFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusFound)
	}
}
