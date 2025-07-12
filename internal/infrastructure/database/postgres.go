package database

import (
	"fmt"
	"time"

	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/config"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresClient wraps the GORM database connection
type PostgresClient struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// NewPostgresClient creates a new PostgreSQL connection using GORM
func NewPostgresClient(cfg *config.Config, log *logrus.Logger) (*PostgresClient, error) {
	dsn := cfg.GetDatabaseDSN()

	// Configure GORM logger
	var gormLogger logger.Interface
	if cfg.App.Environment == "production" {
		gormLogger = logger.Default.LogMode(logger.Silent)
	} else {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	log.Info("Connected to PostgreSQL successfully")

	return &PostgresClient{
		db:     db,
		logger: log,
	}, nil
}

// GetDB returns the underlying GORM database instance
func (p *PostgresClient) GetDB() *gorm.DB {
	return p.db
}

// Close closes the database connection
func (p *PostgresClient) Close() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping tests the database connection
func (p *PostgresClient) Ping() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Transaction methods
func (p *PostgresClient) BeginTransaction() *gorm.DB {
	return p.db.Begin()
}

func (p *PostgresClient) Rollback(tx *gorm.DB) {
	tx.Rollback()
}

func (p *PostgresClient) Commit(tx *gorm.DB) error {
	return tx.Commit().Error
}

// Health check method
func (p *PostgresClient) HealthCheck() error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Stats returns database connection statistics
func (p *PostgresClient) Stats() map[string]interface{} {
	sqlDB, err := p.db.DB()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"max_open_connections": stats.MaxOpenConnections,
		"open_connections":     stats.OpenConnections,
		"in_use":               stats.InUse,
		"idle":                 stats.Idle,
		"wait_count":           stats.WaitCount,
		"wait_duration":        stats.WaitDuration.String(),
		"max_idle_closed":      stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed":  stats.MaxLifetimeClosed,
	}
}
