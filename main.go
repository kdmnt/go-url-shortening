package main

import (
	"flag"
	"go-url-shortening/config"
	"go-url-shortening/server"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	cfg    *config.Config
)

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic("Failed to initialize zap logger: " + err.Error())
	}
	cfg = config.DefaultConfig()
}

func parseFlags() {
	disableRateLimit := flag.Bool("disable-rate-limit", false, "Disable rate limiting for performance testing")
	flag.Parse()
	cfg.DisableRateLimit = *disableRateLimit
}

func main() {
	defer logger.Sync()

	parseFlags()

	logger.Info("Starting URL Shortener application...")
	if err := server.Run(logger, cfg); err != nil {
		logger.Fatal("Application error", zap.Error(err))
	}
	logger.Info("URL Shortener application stopped.")
}
