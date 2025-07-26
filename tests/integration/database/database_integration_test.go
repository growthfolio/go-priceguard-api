package database_integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/adapters/repository"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseIntegrationTestSuite struct {
	suite.Suite
	db  *gorm.DB
	ctx context.Context
}

func (suite *DatabaseIntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Setup in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	suite.Require().NoError(err)

	// Auto-migrate all tables
	err = db.AutoMigrate(
		&entities.User{},
		&entities.UserSettings{},
		&entities.Alert{},
		&entities.Notification{},
		&entities.CryptoCurrency{},
		&entities.PriceHistory{},
		&entities.TechnicalIndicator{},
		&entities.Session{},
	)
	suite.Require().NoError(err)

	suite.db = db
}

func (suite *DatabaseIntegrationTestSuite) TearDownSuite() {
	if suite.db != nil {
		sqlDB, err := suite.db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

func (suite *DatabaseIntegrationTestSuite) SetupTest() {
	// Clean all tables before each test
	suite.cleanupTables()
}

func (suite *DatabaseIntegrationTestSuite) cleanupTables() {
	tables := []string{
		"sessions", "technical_indicators", "price_histories",
		"crypto_currencies", "notifications", "alerts",
		"user_settings", "users",
	}

	for _, table := range tables {
		suite.db.Exec("DELETE FROM " + table)
	}
}

func (suite *DatabaseIntegrationTestSuite) TestMigrations() {
	// Test that all tables exist and have correct structure
	var tableNames []string

	// Get all table names
	err := suite.db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tableNames).Error
	suite.NoError(err)

	expectedTables := []string{
		"users", "user_settings", "alerts", "notifications",
		"crypto_currencies", "price_histories", "technical_indicators", "sessions",
	}

	for _, expectedTable := range expectedTables {
		suite.Contains(tableNames, expectedTable, "Table %s should exist", expectedTable)
	}
}

func (suite *DatabaseIntegrationTestSuite) TestUserConstraints() {
	userRepo := repository.NewUserRepository(suite.db)

	// Test unique constraint on google_id
	user1 := &entities.User{
		GoogleID: "google123",
		Email:    "test1@example.com",
		Name:     "Test User 1",
	}

	err := userRepo.Create(suite.ctx, user1)
	suite.NoError(err)

	// Try to create another user with same GoogleID
	user2 := &entities.User{
		GoogleID: "google123", // Same GoogleID
		Email:    "test2@example.com",
		Name:     "Test User 2",
	}

	err = userRepo.Create(suite.ctx, user2)
	suite.Error(err, "Should fail due to unique constraint on google_id")

	// Test unique constraint on email
	user3 := &entities.User{
		GoogleID: "google456",
		Email:    "test1@example.com", // Same email as user1
		Name:     "Test User 3",
	}

	err = userRepo.Create(suite.ctx, user3)
	suite.Error(err, "Should fail due to unique constraint on email")
}

func (suite *DatabaseIntegrationTestSuite) TestAlertConstraints() {
	// First create a user
	userRepo := repository.NewUserRepository(suite.db)
	user := &entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := userRepo.Create(suite.ctx, user)
	suite.NoError(err)

	alertRepo := repository.NewAlertRepository(suite.db)

	// Test foreign key constraint
	alert := &entities.Alert{
		UserID:        uuid.New(), // Non-existent user ID
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}

	err = alertRepo.Create(suite.ctx, alert)
	suite.Error(err, "Should fail due to foreign key constraint")

	// Test with valid user ID
	alert.UserID = user.ID
	err = alertRepo.Create(suite.ctx, alert)
	suite.NoError(err)

	// Test check constraints (if any)
	invalidAlert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "", // Empty symbol should be invalid
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}

	err = alertRepo.Create(suite.ctx, invalidAlert)
	suite.Error(err, "Should fail due to empty symbol")
}

func (suite *DatabaseIntegrationTestSuite) TestTransactions() {
	// Test transaction rollback
	err := suite.db.Transaction(func(tx *gorm.DB) error {
		// Create user in transaction
		user := &entities.User{
			GoogleID: "google123",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		err := tx.Create(user).Error
		if err != nil {
			return err
		}

		// Force an error to test rollback
		return assert.AnError
	})

	suite.Error(err, "Transaction should fail")

	// Verify user was not created (transaction rolled back)
	var count int64
	suite.db.Model(&entities.User{}).Count(&count)
	suite.Equal(int64(0), count, "User should not exist due to rollback")

	// Test successful transaction
	var createdUser *entities.User
	err = suite.db.Transaction(func(tx *gorm.DB) error {
		user := &entities.User{
			GoogleID: "google123",
			Email:    "test@example.com",
			Name:     "Test User",
		}

		err := tx.Create(user).Error
		if err != nil {
			return err
		}

		createdUser = user
		return nil
	})

	suite.NoError(err)
	suite.NotNil(createdUser)
	suite.NotEqual(uuid.Nil, createdUser.ID)

	// Verify user was created
	suite.db.Model(&entities.User{}).Count(&count)
	suite.Equal(int64(1), count)
}

func (suite *DatabaseIntegrationTestSuite) TestIndexPerformance() {
	// Create test data
	userRepo := repository.NewUserRepository(suite.db)
	alertRepo := repository.NewAlertRepository(suite.db)

	// Create a user
	user := &entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := userRepo.Create(suite.ctx, user)
	suite.NoError(err)

	// Create multiple alerts
	numAlerts := 1000
	for i := 0; i < numAlerts; i++ {
		alert := &entities.Alert{
			UserID:        user.ID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   float64(50000 + i),
			Timeframe:     "1h",
			Enabled:       i%2 == 0, // Half enabled, half disabled
			NotifyVia:     []string{"app"},
		}

		err := alertRepo.Create(suite.ctx, alert)
		suite.NoError(err)
	}

	// Test index performance on common queries
	start := time.Now()

	// Query by user_id (should be fast with index)
	alerts, err := alertRepo.GetByUserID(suite.ctx, user.ID, 100, 0)
	duration1 := time.Since(start)

	suite.NoError(err)
	suite.Greater(len(alerts), 0)
	suite.Less(duration1, 100*time.Millisecond, "Query by user_id should be fast")

	start = time.Now()

	// Query enabled alerts (should be fast with index)
	enabledAlerts, err := alertRepo.GetEnabled(suite.ctx)
	duration2 := time.Since(start)

	suite.NoError(err)
	suite.Greater(len(enabledAlerts), 0)
	suite.Less(duration2, 100*time.Millisecond, "Query by enabled should be fast")

	start = time.Now()

	// Query by symbol (should be fast with index)
	symbolAlerts, err := alertRepo.GetBySymbol(suite.ctx, "BTCUSDT")
	duration3 := time.Since(start)

	suite.NoError(err)
	suite.Equal(numAlerts, len(symbolAlerts))
	suite.Less(duration3, 100*time.Millisecond, "Query by symbol should be fast")
}

func (suite *DatabaseIntegrationTestSuite) TestConcurrentAccess() {
	userRepo := repository.NewUserRepository(suite.db)
	alertRepo := repository.NewAlertRepository(suite.db)

	// Create a user
	user := &entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := userRepo.Create(suite.ctx, user)
	suite.NoError(err)

	// Test concurrent alert creation
	numGoroutines := 10
	numAlertsPerGoroutine := 10

	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < numAlertsPerGoroutine; j++ {
				alert := &entities.Alert{
					UserID:        user.ID,
					Symbol:        "BTCUSDT",
					AlertType:     "price",
					ConditionType: "above",
					TargetValue:   float64(50000 + goroutineID*1000 + j),
					Timeframe:     "1h",
					Enabled:       true,
					NotifyVia:     []string{"app"},
				}

				err := alertRepo.Create(suite.ctx, alert)
				if err != nil {
					errChan <- err
					return
				}
			}
			errChan <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		suite.NoError(err)
	}

	// Verify all alerts were created
	alerts, err := alertRepo.GetByUserID(suite.ctx, user.ID, 1000, 0)
	suite.NoError(err)
	suite.Equal(numGoroutines*numAlertsPerGoroutine, len(alerts))
}

func (suite *DatabaseIntegrationTestSuite) TestConnectionPooling() {
	// Test that database handles multiple connections properly
	sqlDB, err := suite.db.DB()
	suite.NoError(err)

	// Test connection stats
	stats := sqlDB.Stats()
	suite.GreaterOrEqual(stats.OpenConnections, 0)
	suite.GreaterOrEqual(stats.InUse, 0)
	suite.GreaterOrEqual(stats.Idle, 0)

	// Configure connection pool settings for testing
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test that settings are applied
	newStats := sqlDB.Stats()
	suite.LessOrEqual(newStats.OpenConnections, 10)
	suite.LessOrEqual(newStats.Idle, 5)
}

func (suite *DatabaseIntegrationTestSuite) TestDataIntegrity() {
	// Test cascading deletes and data integrity
	userRepo := repository.NewUserRepository(suite.db)
	alertRepo := repository.NewAlertRepository(suite.db)
	notificationRepo := repository.NewNotificationRepository(suite.db)

	// Create user
	user := &entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := userRepo.Create(suite.ctx, user)
	suite.NoError(err)

	// Create alert
	alert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}
	err = alertRepo.Create(suite.ctx, alert)
	suite.NoError(err)

	// Create notification
	notification := &entities.Notification{
		UserID:           user.ID,
		AlertID:          &alert.ID,
		Title:            "Test Notification",
		Message:          "Test message",
		NotificationType: "alert_triggered",
		CreatedAt:        time.Now(),
	}
	err = notificationRepo.Create(suite.ctx, notification)
	suite.NoError(err)

	// Verify relationships
	var alertCount int64
	suite.db.Model(&entities.Alert{}).Where("user_id = ?", user.ID).Count(&alertCount)
	suite.Equal(int64(1), alertCount)

	var notificationCount int64
	suite.db.Model(&entities.Notification{}).Where("user_id = ?", user.ID).Count(&notificationCount)
	suite.Equal(int64(1), notificationCount)

	// Test that foreign key constraints prevent orphaned records
	err = suite.db.Delete(user).Error
	if err == nil {
		// If cascade delete is enabled, verify related records are deleted
		suite.db.Model(&entities.Alert{}).Where("user_id = ?", user.ID).Count(&alertCount)
		suite.Equal(int64(0), alertCount)

		suite.db.Model(&entities.Notification{}).Where("user_id = ?", user.ID).Count(&notificationCount)
		suite.Equal(int64(0), notificationCount)
	} else {
		// If cascade delete is not enabled, deletion should fail due to foreign key constraint
		suite.Error(err, "Should fail due to foreign key constraint")
	}
}

func (suite *DatabaseIntegrationTestSuite) TestQueryOptimization() {
	// Test various query patterns for optimization
	userRepo := repository.NewUserRepository(suite.db)
	alertRepo := repository.NewAlertRepository(suite.db)

	// Create test data
	user := &entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := userRepo.Create(suite.ctx, user)
	suite.NoError(err)

	// Create alerts with various symbols
	symbols := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT", "DOTUSDT", "LINKUSDT"}
	for i, symbol := range symbols {
		for j := 0; j < 20; j++ {
			alert := &entities.Alert{
				UserID:        user.ID,
				Symbol:        symbol,
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   float64(1000 + i*1000 + j),
				Timeframe:     "1h",
				Enabled:       j%2 == 0,
				NotifyVia:     []string{"app"},
			}

			err := alertRepo.Create(suite.ctx, alert)
			suite.NoError(err)
		}
	}

	// Test various query patterns
	start := time.Now()

	// 1. Simple select with WHERE clause
	var alerts []entities.Alert
	err = suite.db.Where("symbol = ? AND enabled = ?", "BTCUSDT", true).Find(&alerts).Error
	duration1 := time.Since(start)
	suite.NoError(err)
	suite.Less(duration1, 50*time.Millisecond, "Simple WHERE query should be fast")

	start = time.Now()

	// 2. Query with JOIN (if applicable)
	var results []struct {
		AlertID   uuid.UUID
		AlertType string
		UserEmail string
	}
	err = suite.db.Table("alerts").
		Select("alerts.id as alert_id, alerts.alert_type, users.email as user_email").
		Joins("JOIN users ON alerts.user_id = users.id").
		Where("alerts.enabled = ?", true).
		Find(&results).Error
	duration2 := time.Since(start)
	suite.NoError(err)
	suite.Less(duration2, 100*time.Millisecond, "JOIN query should be reasonably fast")

	start = time.Now()

	// 3. Aggregation query
	var symbolCounts []struct {
		Symbol string
		Count  int64
	}
	err = suite.db.Table("alerts").
		Select("symbol, COUNT(*) as count").
		Group("symbol").
		Find(&symbolCounts).Error
	duration3 := time.Since(start)
	suite.NoError(err)
	suite.Equal(len(symbols), len(symbolCounts))
	suite.Less(duration3, 100*time.Millisecond, "GROUP BY query should be reasonably fast")
}

func TestDatabaseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseIntegrationTestSuite))
}

// Additional database integration tests

func TestDatabaseMigrationRollback(t *testing.T) {
	// Test migration rollback scenarios
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	// Test partial migration
	err = db.AutoMigrate(&entities.User{})
	assert.NoError(t, err)

	// Verify table exists
	var tableNames []string
	err = db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tableNames).Error
	assert.NoError(t, err)
	assert.Contains(t, tableNames, "users")

	// Test that we can drop and recreate
	err = db.Migrator().DropTable(&entities.User{})
	assert.NoError(t, err)

	err = db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tableNames).Error
	assert.NoError(t, err)
	assert.NotContains(t, tableNames, "users")
}

func TestDatabasePerformanceUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	err = db.AutoMigrate(&entities.User{}, &entities.Alert{})
	assert.NoError(t, err)

	userRepo := repository.NewUserRepository(db)
	alertRepo := repository.NewAlertRepository(db)
	ctx := context.Background()

	// Create user
	user := &entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err = userRepo.Create(ctx, user)
	assert.NoError(t, err)

	// Performance test: Create many alerts rapidly
	start := time.Now()
	numAlerts := 10000

	for i := 0; i < numAlerts; i++ {
		alert := &entities.Alert{
			UserID:        user.ID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   float64(50000 + i),
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		}

		err := alertRepo.Create(ctx, alert)
		assert.NoError(t, err)
	}

	duration := time.Since(start)
	alertsPerSecond := float64(numAlerts) / duration.Seconds()

	t.Logf("Created %d alerts in %v (%.2f alerts/second)", numAlerts, duration, alertsPerSecond)
	assert.Greater(t, alertsPerSecond, 100.0, "Should be able to create at least 100 alerts/second")
}
