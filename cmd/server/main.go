package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	httphandler "github.com/growthfolio/go-priceguard-api/internal/adapters/http"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/config"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/database"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup loggers
	logger := setupLogger(cfg)
	zapLogger, err := infrastructure.NewDefaultLogger()
	if err != nil {
		logger.Fatalf("Failed to initialize zap logger: %v", err)
	}
	defer zapLogger.Sync()

	logger.Info("Starting PriceGuard API server...")
	zapLogger.Info("Zap logger initialized successfully")

	// Initialize tracing
	tracingManager, err := infrastructure.NewDefaultTracingManager(zapLogger)
	if err != nil {
		logger.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer func() {
		if tracingManager != nil {
			if err := tracingManager.Shutdown(context.Background()); err != nil {
				logger.Errorf("Failed to shutdown tracing: %v", err)
			}
		}
	}()

	// Initialize database connections
	dbManager, err := database.NewManager(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize database connections: %v", err)
	}
	defer dbManager.Close()

	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize Gin router
	router := gin.New()

	// Setup API routes and WebSocket (includes health endpoints)
	routerDeps := &httphandler.RouterDependencies{
		Config:         cfg,
		Logger:         logger,
		ZapLogger:      zapLogger,
		DBManager:      dbManager,
		RedisClient:    dbManager.GetRedis().GetClient(),
		TracingManager: tracingManager,
	}
	wsManager := httphandler.SetupRoutes(router, routerDeps)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Server starting on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Stop WebSocket worker
	if wsManager != nil && wsManager.Worker != nil {
		wsManager.Worker.Stop()
	}

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

// setupLogger configures the application logger
func setupLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	switch cfg.App.LogLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// Set formatter
	if cfg.App.Environment == "production" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return logger
}
