package database

import (
	"context"
	"testing"
	"time"

	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseManager(t *testing.T) {
	// Skip integration tests if not in integration mode
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Create test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "password",
			Name:     "priceguard_test",
			SSLMode:  "disable",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       1, // Use different DB for tests
		},
		App: config.AppConfig{
			Environment: "test",
			LogLevel:    "debug",
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test database manager creation
	manager, err := NewManager(cfg, logger)
	if err != nil {
		t.Skipf("Skipping test due to database connection error: %v", err)
	}
	defer manager.Close()

	// Test health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health := manager.HealthCheck(ctx)
	require.NotNil(t, health)

	// Test PostgreSQL health
	if pgHealth, ok := health["postgres"].(map[string]interface{}); ok {
		assert.Equal(t, "healthy", pgHealth["status"])
	}

	// Test Redis health
	if redisHealth, ok := health["redis"].(map[string]interface{}); ok {
		assert.Equal(t, "healthy", redisHealth["status"])
	}

	// Test IsHealthy method
	assert.True(t, manager.IsHealthy(ctx))
}

func TestRedisOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       1,
		},
	}

	logger := logrus.New()
	redis, err := NewRedisClient(cfg, logger)
	if err != nil {
		t.Skipf("Skipping Redis test due to connection error: %v", err)
	}
	defer redis.Close()

	ctx := context.Background()

	// Test basic operations
	err = redis.SetCryptoData(ctx, "BTCUSDT", `{"price": 50000}`, time.Minute)
	assert.NoError(t, err)

	data, err := redis.GetCryptoData(ctx, "BTCUSDT")
	assert.NoError(t, err)
	assert.Equal(t, `{"price": 50000}`, data)

	// Test session operations
	err = redis.SetSession(ctx, "test-session", "user-123", time.Hour)
	assert.NoError(t, err)

	userID, err := redis.GetSession(ctx, "test-session")
	assert.NoError(t, err)
	assert.Equal(t, "user-123", userID)

	// Clean up
	redis.DeleteSession(ctx, "test-session")
}
