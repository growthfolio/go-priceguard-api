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

	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/config"
	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/database"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger
	logger := setupLogger(cfg)
	logger.Info("Starting PriceGuard API server...")

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

	// Add basic middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Basic health check endpoint
	router.GET("/health", func(c *gin.Context) {
		// Check database health
		dbHealth := dbManager.HealthCheck(c.Request.Context())
		
		response := gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
			"database":  dbHealth,
		}

		// Return 503 if any database is unhealthy
		if !dbManager.IsHealthy(c.Request.Context()) {
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}

		c.JSON(http.StatusOK, response)
	})

	// Basic API info endpoint
	router.GET("/api/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name":        "PriceGuard API",
			"version":     "1.0.0",
			"description": "Backend API for PriceGuard cryptocurrency monitoring application",
			"environment": cfg.App.Environment,
		})
	})

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
