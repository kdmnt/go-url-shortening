package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 10, cfg.RateLimit, "RateLimit should be 10")
	assert.Equal(t, time.Second, cfg.RatePeriod, "RatePeriod should be 1 second")
	assert.Equal(t, 5*time.Second, cfg.RequestTimeout, "RequestTimeout should be 5 seconds")
	assert.Equal(t, 3000, cfg.ServerPort, "ServerPort should be 3000")
	assert.False(t, cfg.DisableRateLimit, "DisableRateLimit should be false")
}
