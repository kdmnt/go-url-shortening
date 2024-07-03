package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"go-url-shortening/config"
	"go-url-shortening/server"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
}

func main() {
	disableRateLimit := flag.Bool("disable-rate-limit", false, "Disable rate limiting for performance testing")
	flag.Parse()

	cfg := config.DefaultConfig()
	cfg.DisableRateLimit = *disableRateLimit

	logger.Info("Starting URL Shortener application...")
	if err := server.Run(logger, cfg); err != nil {
		logger.WithError(err).Fatal("Application error")
	}
	logger.Info("URL Shortener application stopped.")
}
