// Package config provides configuration settings for the URL shortener service.
package config

import "time"

// Config holds the configuration settings for the application.
type Config struct {
	RateLimit        int
	RatePeriod       time.Duration
	RequestTimeout   time.Duration
	ServerPort       int
	DisableRateLimit bool
}

// DefaultConfig returns the default configuration settings.
// Caveat: These could be loaded from Env Vars in a production setting
func DefaultConfig() *Config {
	return &Config{
		RateLimit:        10,
		RatePeriod:       time.Second,
		RequestTimeout:   5 * time.Second,
		ServerPort:       3000,
		DisableRateLimit: false,
	}
}
