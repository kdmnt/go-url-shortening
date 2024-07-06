package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInit(t *testing.T) {
	// Assert that logger is properly initialized
	assert.NotNil(t, logger)
	assert.IsType(t, (*zap.Logger)(nil), logger)

	// Check if the log level is set to InfoLevel
	assert.True(t, logger.Core().Enabled(zapcore.InfoLevel))
}
