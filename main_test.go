package main

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"go-url-shortening/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	// Assert that logger is properly initialized
	assert.NotNil(t, logger)
	assert.IsType(t, (*zap.Logger)(nil), logger)

	// Check if the log level is set to InfoLevel
	assert.True(t, logger.Core().Enabled(zapcore.InfoLevel))
}

func TestConfigInitialization(t *testing.T) {
	// Check if cfg is properly initialized
	assert.NotNil(t, cfg)
	assert.Equal(t, config.DefaultConfig(), cfg)

	// Verify that modifying cfg doesn't affect the default config
	originalRateLimit := cfg.RateLimit
	cfg.RateLimit = 20
	assert.NotEqual(t, config.DefaultConfig().RateLimit, cfg.RateLimit)

	// Reset cfg to original state
	cfg.RateLimit = originalRateLimit
}

func TestDisableRateLimitFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"With flag", []string{"-disable-rate-limit"}, true},
		{"Without flag", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags and cfg for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			cfg = config.DefaultConfig()

			// Set up test arguments
			os.Args = append([]string{"cmd"}, tt.args...)

			// Parse flags
			parseFlags()

			// Check if DisableRateLimit is set correctly
			assert.Equal(t, tt.expected, cfg.DisableRateLimit, "DisableRateLimit should be %v when flag is %v", tt.expected, tt.args)
		})
	}
}
