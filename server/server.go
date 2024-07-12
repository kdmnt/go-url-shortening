// Package server provides the main server setup and run functionality for the URL shortening service.
package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go-url-shortening/config"
	"go-url-shortening/handlers"
	"go-url-shortening/services"
	"go-url-shortening/storage"
	"go.uber.org/zap"
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

	var wg sync.WaitGroup
	errChan := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startServer(ctx, server, logger); err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
	}()

	// Ensure the goroutine is cleaned up
	defer func() {
		cancel()
		wg.Wait()
	}()

	select {
	case err := <-errChan:
		cancel()
		wg.Wait()
		return err
	case <-time.After(100 * time.Millisecond):
		err := waitForShutdown(ctx, server, logger)
		wg.Wait()
		return err
	}
}

// setupURLHandler creates and configures the URL handler with necessary dependencies.
// It returns the configured handler or an error if setup fails.
func setupURLHandler(ctx context.Context, cfg *config.Config, store storage.Storage, logger *zap.Logger) (handlers.URLHandlerInterface, error) {
	handlerCtx, cancel := context.WithTimeout(ctx, cfg.RequestTimeout)
	defer cancel()

	urlService := services.NewURLService(store)

	handler, err := handlers.NewURLHandler(handlerCtx, urlService, cfg, logger)
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
		Addr:    ":" + strconv.Itoa(cfg.ServerPort),
		Handler: router,
	}
}

// startServer begins listening and serving HTTP requests.
// It logs any errors that occur during server operation.
func startServer(ctx context.Context, srv *http.Server, logger *zap.Logger) error {
	logger.Debug("Starting server", zap.String("address", srv.Addr))

	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Debug("Context cancelled, stopping server")
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Error("Error shutting down server", zap.Error(err))
		}
		return ctx.Err()
	case err := <-errChan:
		logger.Error("Server error", zap.Error(err))
		return err
	}
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
