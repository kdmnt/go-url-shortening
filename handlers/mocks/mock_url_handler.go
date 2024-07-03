package mocks

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
)

type MockURLHandler struct {
	mock.Mock
}

func (m *MockURLHandler) CreateShortURL(c *gin.Context) {
	m.Called(c)
}

func (m *MockURLHandler) GetURLData(c *gin.Context) {
	m.Called(c)
}

func (m *MockURLHandler) UpdateURL(c *gin.Context) {
	m.Called(c)
}

func (m *MockURLHandler) DeleteURL(c *gin.Context) {
	m.Called(c)
}

func (m *MockURLHandler) HealthCheck(c *gin.Context) {
	m.Called(c)
}

func (m *MockURLHandler) RedirectURL(c *gin.Context) {
	m.Called(c)
}

func (m *MockURLHandler) RateLimitMiddleware() gin.HandlerFunc {
	args := m.Called()
	return args.Get(0).(gin.HandlerFunc)
}
