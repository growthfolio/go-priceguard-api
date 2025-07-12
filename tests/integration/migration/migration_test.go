package migration_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MigrationTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (suite *MigrationTestSuite) SetupTest() {
	// Create a new in-memory database for each test
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	suite.Require().NoError(err)
	suite.db = db
}

func (suite *MigrationTestSuite) TearDownTest() {
	if suite.db != nil {
		sqlDB, err := suite.db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

func (suite *MigrationTestSuite) TestMigrationOrder() {
	// Test that migrations can be applied in the correct order

	// 1. First migrate User (no dependencies)
	err := suite.db.AutoMigrate(&entities.User{})
	suite.NoError(err)

	// Verify User table exists
	suite.assertTrue(suite.db.Migrator().HasTable(&entities.User{}))

	// 2. Migrate UserSettings (depends on User)
	err = suite.db.AutoMigrate(&entities.UserSettings{})
	suite.NoError(err)

	suite.assertTrue(suite.db.Migrator().HasTable(&entities.UserSettings{}))

	// 3. Migrate CryptoCurrency (no dependencies)
	err = suite.db.AutoMigrate(&entities.CryptoCurrency{})
	suite.NoError(err)

	suite.assertTrue(suite.db.Migrator().HasTable(&entities.CryptoCurrency{}))

	// 4. Migrate Alert (depends on User)
	err = suite.db.AutoMigrate(&entities.Alert{})
	suite.NoError(err)

	suite.assertTrue(suite.db.Migrator().HasTable(&entities.Alert{}))

	// 5. Migrate Notification (depends on User and Alert)
	err = suite.db.AutoMigrate(&entities.Notification{})
	suite.NoError(err)

	suite.assertTrue(suite.db.Migrator().HasTable(&entities.Notification{}))

	// 6. Migrate remaining entities
	err = suite.db.AutoMigrate(
		&entities.PriceHistory{},
		&entities.TechnicalIndicator{},
		&entities.Session{},
	)
	suite.NoError(err)

	// Verify all tables exist
	expectedTables := []interface{}{
		&entities.User{},
		&entities.UserSettings{},
		&entities.CryptoCurrency{},
		&entities.Alert{},
		&entities.Notification{},
		&entities.PriceHistory{},
		&entities.TechnicalIndicator{},
		&entities.Session{},
	}

	for _, table := range expectedTables {
		suite.assertTrue(suite.db.Migrator().HasTable(table))
	}
}

func (suite *MigrationTestSuite) TestMigrationColumns() {
	// Test that all columns are created correctly
	err := suite.db.AutoMigrate(&entities.User{})
	suite.NoError(err)

	// Check User columns
	userColumns := []string{"id", "google_id", "email", "name", "avatar_url", "created_at", "updated_at"}
	for _, column := range userColumns {
		suite.assertTrue(suite.db.Migrator().HasColumn(&entities.User{}, column))
	}

	// Test Alert table
	err = suite.db.AutoMigrate(&entities.Alert{})
	suite.NoError(err)

	alertColumns := []string{
		"id", "user_id", "symbol", "alert_type", "condition_type",
		"target_value", "timeframe", "enabled", "notify_via",
		"triggered_at", "created_at", "updated_at",
	}
	for _, column := range alertColumns {
		suite.assertTrue(suite.db.Migrator().HasColumn(&entities.Alert{}, column))
	}

	// Test Notification table
	err = suite.db.AutoMigrate(&entities.Notification{})
	suite.NoError(err)

	notificationColumns := []string{
		"id", "user_id", "alert_id", "title", "message",
		"notification_type", "read_at", "created_at",
	}
	for _, column := range notificationColumns {
		suite.assertTrue(suite.db.Migrator().HasColumn(&entities.Notification{}, column))
	}
}

func (suite *MigrationTestSuite) TestMigrationIndexes() {
	// Test that indexes are created correctly
	err := suite.db.AutoMigrate(&entities.User{}, &entities.Alert{}, &entities.Notification{})
	suite.NoError(err)

	// Test User indexes
	suite.assertTrue(suite.db.Migrator().HasIndex(&entities.User{}, "email"))
	suite.assertTrue(suite.db.Migrator().HasIndex(&entities.User{}, "google_id"))

	// Test Alert indexes
	suite.assertTrue(suite.db.Migrator().HasIndex(&entities.Alert{}, "user_id"))
	suite.assertTrue(suite.db.Migrator().HasIndex(&entities.Alert{}, "symbol"))

	// Test Notification indexes
	suite.assertTrue(suite.db.Migrator().HasIndex(&entities.Notification{}, "user_id"))
	suite.assertTrue(suite.db.Migrator().HasIndex(&entities.Notification{}, "alert_id"))
}

func (suite *MigrationTestSuite) TestMigrationConstraints() {
	// Test that constraints are applied correctly
	err := suite.db.AutoMigrate(&entities.User{}, &entities.Alert{})
	suite.NoError(err)

	// Test unique constraints by trying to insert duplicates
	user1 := entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	err = suite.db.Create(&user1).Error
	suite.NoError(err)

	// Try to create another user with same email
	user2 := entities.User{
		GoogleID: "google456",
		Email:    "test@example.com", // Same email
		Name:     "Another User",
	}

	err = suite.db.Create(&user2).Error
	suite.Error(err, "Should fail due to unique constraint on email")

	// Try to create another user with same GoogleID
	user3 := entities.User{
		GoogleID: "google123", // Same GoogleID
		Email:    "another@example.com",
		Name:     "Yet Another User",
	}

	err = suite.db.Create(&user3).Error
	suite.Error(err, "Should fail due to unique constraint on google_id")
}

func (suite *MigrationTestSuite) TestMigrationRollback() {
	// Test that we can drop tables (simulating rollback)

	// First, create all tables
	err := suite.db.AutoMigrate(
		&entities.User{},
		&entities.Alert{},
		&entities.Notification{},
	)
	suite.NoError(err)

	// Verify tables exist
	suite.assertTrue(suite.db.Migrator().HasTable(&entities.User{}))
	suite.assertTrue(suite.db.Migrator().HasTable(&entities.Alert{}))
	suite.assertTrue(suite.db.Migrator().HasTable(&entities.Notification{}))

	// Drop tables in reverse order (due to foreign key constraints)
	err = suite.db.Migrator().DropTable(&entities.Notification{})
	suite.NoError(err)
	suite.assertFalse(suite.db.Migrator().HasTable(&entities.Notification{}))

	err = suite.db.Migrator().DropTable(&entities.Alert{})
	suite.NoError(err)
	suite.assertFalse(suite.db.Migrator().HasTable(&entities.Alert{}))

	err = suite.db.Migrator().DropTable(&entities.User{})
	suite.NoError(err)
	suite.assertFalse(suite.db.Migrator().HasTable(&entities.User{}))
}

func (suite *MigrationTestSuite) TestMigrationModification() {
	// Test modifying existing tables (adding/removing columns)

	// Create initial User table
	err := suite.db.AutoMigrate(&entities.User{})
	suite.NoError(err)

	// Verify initial columns
	suite.assertTrue(suite.db.Migrator().HasColumn(&entities.User{}, "email"))
	suite.assertTrue(suite.db.Migrator().HasColumn(&entities.User{}, "name"))

	// Test adding a column (simplified for GORM compatibility)
	type UserWithTestColumn struct {
		entities.User
		TestColumn string `gorm:"type:varchar(255)"`
	}

	err = suite.db.AutoMigrate(&UserWithTestColumn{})
	suite.NoError(err)
	suite.assertTrue(suite.db.Migrator().HasColumn(&UserWithTestColumn{}, "test_column"))

	// Test dropping a column
	err = suite.db.Migrator().DropColumn(&UserWithTestColumn{}, "test_column")
	suite.NoError(err)
	suite.assertFalse(suite.db.Migrator().HasColumn(&UserWithTestColumn{}, "test_column"))
}

func (suite *MigrationTestSuite) TestMigrationWithData() {
	// Test that migrations preserve existing data

	// Create User table and insert data
	err := suite.db.AutoMigrate(&entities.User{})
	suite.NoError(err)

	user := entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	err = suite.db.Create(&user).Error
	suite.NoError(err)

	// Run migration again (should be idempotent)
	err = suite.db.AutoMigrate(&entities.User{})
	suite.NoError(err)

	// Verify data still exists
	var retrievedUser entities.User
	err = suite.db.Where("email = ?", "test@example.com").First(&retrievedUser).Error
	suite.NoError(err)
	suite.Equal("google123", retrievedUser.GoogleID)
	suite.Equal("Test User", retrievedUser.Name)
}

func (suite *MigrationTestSuite) TestMigrationForeignKeys() {
	// Test foreign key constraints
	err := suite.db.AutoMigrate(&entities.User{}, &entities.Alert{})
	suite.NoError(err)

	// Create user first
	user := entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	err = suite.db.Create(&user).Error
	suite.NoError(err)

	// Create alert with valid user_id
	alert := entities.Alert{
		UserID:        user.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}

	err = suite.db.Create(&alert).Error
	suite.NoError(err)

	// Verify foreign key relationship
	var retrievedAlert entities.Alert
	err = suite.db.Preload("User").Where("id = ?", alert.ID).First(&retrievedAlert).Error
	suite.NoError(err)
	suite.Equal(user.ID, retrievedAlert.User.ID)
	suite.Equal(user.Email, retrievedAlert.User.Email)
}

// Helper methods
func (suite *MigrationTestSuite) assertTrue(condition bool) {
	suite.True(condition)
}

func (suite *MigrationTestSuite) assertFalse(condition bool) {
	suite.False(condition)
}

func TestMigrationTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

// Additional migration tests

func TestMigrationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	// Test migration performance with large data set
	err = db.AutoMigrate(&entities.User{})
	assert.NoError(t, err)

	// Insert large number of users
	users := make([]entities.User, 10000)
	for i := 0; i < 10000; i++ {
		users[i] = entities.User{
			GoogleID: fmt.Sprintf("google%d", i),
			Email:    fmt.Sprintf("user%d@example.com", i),
			Name:     fmt.Sprintf("User %d", i),
		}
	}

	err = db.CreateInBatches(users, 1000).Error
	assert.NoError(t, err)

	// Test migration with existing data
	start := time.Now()
	err = db.AutoMigrate(&entities.Alert{})
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, duration, 5*time.Second, "Migration should complete within 5 seconds")

	// Verify data integrity after migration
	var userCount int64
	db.Model(&entities.User{}).Count(&userCount)
	assert.Equal(t, int64(10000), userCount)
}

func TestMigrationDataTypes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)

	// Test that data types are correctly mapped
	err = db.AutoMigrate(&entities.PriceHistory{})
	assert.NoError(t, err)

	// Test decimal precision for price fields
	priceHistory := entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		OpenPrice:  50000.12345678,
		HighPrice:  51000.87654321,
		LowPrice:   49000.11111111,
		ClosePrice: 50500.99999999,
		Volume:     1000000.12345678,
		Timestamp:  time.Now(),
	}

	err = db.Create(&priceHistory).Error
	assert.NoError(t, err)

	// Retrieve and verify precision
	var retrieved entities.PriceHistory
	err = db.First(&retrieved, priceHistory.ID).Error
	assert.NoError(t, err)

	// Note: SQLite may not preserve all decimal places, but the data should be reasonable
	assert.InDelta(t, priceHistory.OpenPrice, retrieved.OpenPrice, 0.01)
	assert.InDelta(t, priceHistory.ClosePrice, retrieved.ClosePrice, 0.01)
}
