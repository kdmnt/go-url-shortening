package main

import (
	"github.com/sirupsen/logrus"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	// Assert that logger is properly initialized
	assert.NotNil(t, logger)
	assert.IsType(t, &logrus.Logger{}, logger)

	// Check if the formatter is JSONFormatter
	assert.IsType(t, &logrus.JSONFormatter{}, logger.Formatter)

	// Check if the log level is set to InfoLevel
	assert.Equal(t, logrus.InfoLevel, logger.Level)
}
