package main

import (
	"flag"
	"go-url-shortening/config"
	"go-url-shortening/server"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic("Failed to initialize zap logger: " + err.Error())
	}
}

func main() {
	defer logger.Sync()

	disableRateLimit := flag.Bool("disable-rate-limit", false, "Disable rate limiting for performance testing")
	flag.Parse()

	cfg := config.DefaultConfig()
	cfg.DisableRateLimit = *disableRateLimit

	logger.Info("Starting URL Shortener application...")
	if err := server.Run(logger, cfg); err != nil {
		logger.Fatal("Application error", zap.Error(err))
	}
	logger.Info("URL Shortener application stopped.")
}
