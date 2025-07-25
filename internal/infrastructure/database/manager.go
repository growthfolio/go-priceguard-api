package database

import (
	"context"
	"fmt"

	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/config"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Manager coordinates all database connections
type Manager struct {
	Postgres *PostgresClient
	Redis    *RedisClient
	logger   *logrus.Logger
}

// NewManager creates a new database manager with all connections
func NewManager(cfg *config.Config, logger *logrus.Logger) (*Manager, error) {
	// Initialize PostgreSQL connection
	postgres, err := NewPostgresClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgreSQL: %w", err)
	}

	// Initialize Redis connection
	redis, err := NewRedisClient(cfg, logger)
	if err != nil {
		// Close PostgreSQL connection if Redis fails
		postgres.Close()
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}

	return &Manager{
		Postgres: postgres,
		Redis:    redis,
		logger:   logger,
	}, nil
}

// Close closes all database connections
func (m *Manager) Close() error {
	var errors []error

	// Close PostgreSQL
	if err := m.Postgres.Close(); err != nil {
		errors = append(errors, fmt.Errorf("postgres close error: %w", err))
	}

	// Close Redis
	if err := m.Redis.Close(); err != nil {
		errors = append(errors, fmt.Errorf("redis close error: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("database close errors: %v", errors)
	}

	m.logger.Info("All database connections closed successfully")
	return nil
}

// HealthCheck performs health checks on all database connections
func (m *Manager) HealthCheck(ctx context.Context) map[string]interface{} {
	status := map[string]interface{}{
		"postgres": map[string]interface{}{
			"status": "unknown",
		},
		"redis": map[string]interface{}{
			"status": "unknown",
		},
	}

	// Check PostgreSQL
	if err := m.Postgres.HealthCheck(); err != nil {
		status["postgres"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		status["postgres"] = map[string]interface{}{
			"status": "healthy",
			"stats":  m.Postgres.Stats(),
		}
	}

	// Check Redis
	if err := m.Redis.Ping(ctx); err != nil {
		status["redis"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		status["redis"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	return status
}

// IsHealthy returns true if all database connections are healthy
func (m *Manager) IsHealthy(ctx context.Context) bool {
	// Check PostgreSQL
	if err := m.Postgres.HealthCheck(); err != nil {
		m.logger.WithError(err).Error("PostgreSQL health check failed")
		return false
	}

	// Check Redis
	if err := m.Redis.Ping(ctx); err != nil {
		m.logger.WithError(err).Error("Redis health check failed")
		return false
	}

	return true
}

// GetDB returns the GORM database instance for PostgreSQL
func (m *Manager) GetDB() *gorm.DB {
	return m.Postgres.db
}

// GetRedis returns the Redis client
func (m *Manager) GetRedis() *RedisClient {
	return m.Redis
}
