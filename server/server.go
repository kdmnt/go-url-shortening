// Package server provides the main server setup and run functionality for the URL shortening service.
package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"go-url-shortening/config"
	"go-url-shortening/handlers"
	"go-url-shortening/services"
	"go-url-shortening/storage"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Run initializes and starts the server, setting up all necessary components.
// It returns an error if any part of the setup or running process fails.
func Run(logger *zap.Logger, cfg *config.Config) error {
	store := storage.NewInMemoryStorage(1000000, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	urlHandler, err := setupURLHandler(ctx, cfg, store, logger)
	if err != nil {
		return err
	}

	router := setupRouter(urlHandler, cfg)
	server := setupServer(cfg, router)

	errChan := make(chan error, 1)
	go func() {
		errChan <- startServer(server, logger)
	}()

	select {
	case err := <-errChan:
		return err
	case <-time.After(100 * time.Millisecond):
		return waitForShutdown(ctx, server, logger)
	}
}

// setupURLHandler creates and configures the URL handler with necessary dependencies.
// It returns the configured handler or an error if setup fails.
func setupURLHandler(ctx context.Context, cfg *config.Config, store storage.Storage, logger *zap.Logger) (handlers.URLHandlerInterface, error) {
	handlerCtx, cancel := context.WithTimeout(ctx, cfg.RequestTimeout)
	defer cancel()

	urlService := services.NewURLService(store)
	limiter := rate.NewLimiter(rate.Every(cfg.RatePeriod), cfg.RateLimit)

	handler, err := handlers.NewURLHandler(handlerCtx, urlService, cfg, logger, limiter)
	if err != nil {
		logger.Error("Failed to create URL handler", zap.Error(err))
		return nil, err
	}

	logger.Debug("URL handler created successfully")
	return handler, nil
}

// setupRouter creates a new Gin router and registers the application routes.
func setupRouter(urlHandler handlers.URLHandlerInterface, cfg *config.Config) *gin.Engine {
	router := gin.Default()
	handlers.RegisterRoutes(router, urlHandler, cfg)
	return router
}

// setupServer creates and returns a new HTTP server with the given configuration and router.
func setupServer(cfg *config.Config, router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:    cfg.ServerPort,
		Handler: router,
	}
}

// startServer begins listening and serving HTTP requests.
// It logs any errors that occur during server operation.
func startServer(srv *http.Server, logger *zap.Logger) error {
	logger.Debug("Starting server", zap.String("address", srv.Addr))
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Error("Server error", zap.Error(err))
		return err
	}
	logger.Debug("Server stopped")
	return nil
}

// waitForShutdown blocks until the server receives an interrupt signal, then initiates a graceful shutdown.
// It returns an error if the shutdown process fails.
func waitForShutdown(ctx context.Context, srv *http.Server, logger *zap.Logger) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	logger.Info("Received interrupt signal. Initiating server shutdown...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	logger.Info("Server gracefully stopped")
	return nil
}
