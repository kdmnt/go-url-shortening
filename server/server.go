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

	go startServer(server, logger)

	return waitForShutdown(ctx, server, logger)
}

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

func setupRouter(urlHandler handlers.URLHandlerInterface, cfg *config.Config) *gin.Engine {
	router := gin.Default()
	handlers.RegisterRoutes(router, urlHandler, cfg)
	return router
}

func setupServer(cfg *config.Config, router *gin.Engine) *http.Server {
	return &http.Server{
		Addr:    cfg.ServerPort,
		Handler: router,
	}
}

func startServer(srv *http.Server, logger *zap.Logger) {
	logger.Debug("Starting server", zap.String("address", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Server error", zap.Error(err))
	}
	logger.Debug("Server stopped")
}

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
