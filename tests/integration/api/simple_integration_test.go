package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/adapters/repository"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type SimpleIntegrationTestSuite struct {
	suite.Suite
	db        *gorm.DB
	alertRepo repositories.AlertRepository
	userRepo  repositories.UserRepository
	testUser  *entities.User
	ctx       context.Context
}

func (suite *SimpleIntegrationTestSuite) SetupSuite() {
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

	// Setup repositories
	alertRepo := repository.NewAlertRepository(db)
	userRepo := repository.NewUserRepository(db)

	suite.db = db
	suite.alertRepo = alertRepo
	suite.userRepo = userRepo
	suite.ctx = context.Background()

	// Create test user
	suite.createTestUser()
}

func (suite *SimpleIntegrationTestSuite) createTestUser() {
	user := &entities.User{
		GoogleID: "test-google-id",
		Email:    "test@example.com",
		Name:     "Test User",
	}

	err := suite.userRepo.Create(suite.ctx, user)
	suite.Require().NoError(err)

	suite.testUser = user
}

func (suite *SimpleIntegrationTestSuite) TearDownSuite() {
	sqlDB, err := suite.db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

func (suite *SimpleIntegrationTestSuite) TestBasicAlertCRUD_Integration() {
	// Test basic CRUD operations for alerts

	// 1. Create an alert
	alert := &entities.Alert{
		UserID:        suite.testUser.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app"},
	}

	err := suite.alertRepo.Create(suite.ctx, alert)
	assert.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, alert.ID)

	// 2. Read the alert back
	retrievedAlert, err := suite.alertRepo.GetByID(suite.ctx, alert.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), alert.Symbol, retrievedAlert.Symbol)
	assert.Equal(suite.T(), alert.TargetValue, retrievedAlert.TargetValue)

	// 3. Update the alert
	retrievedAlert.TargetValue = 55000.0
	err = suite.alertRepo.Update(suite.ctx, retrievedAlert)
	assert.NoError(suite.T(), err)

	// 4. Verify the update
	updatedAlert, err := suite.alertRepo.GetByID(suite.ctx, alert.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 55000.0, updatedAlert.TargetValue)

	// 5. Delete the alert
	err = suite.alertRepo.Delete(suite.ctx, alert.ID)
	assert.NoError(suite.T(), err)

	// 6. Verify deletion
	_, err = suite.alertRepo.GetByID(suite.ctx, alert.ID)
	assert.Error(suite.T(), err) // Should return error for not found
}

func (suite *SimpleIntegrationTestSuite) TestUserAlertsList_Integration() {
	// Test retrieving alerts by user ID

	// 1. Create multiple alerts for the user
	alerts := []*entities.Alert{
		{
			UserID:        suite.testUser.ID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		},
		{
			UserID:        suite.testUser.ID,
			Symbol:        "ETHUSDT",
			AlertType:     "price",
			ConditionType: "below",
			TargetValue:   3000.0,
			Timeframe:     "1h",
			Enabled:       true,
			NotifyVia:     []string{"app"},
		},
	}

	for _, alert := range alerts {
		err := suite.alertRepo.Create(suite.ctx, alert)
		assert.NoError(suite.T(), err)
	}

	// 2. Retrieve alerts by user ID
	userAlerts, err := suite.alertRepo.GetByUserID(suite.ctx, suite.testUser.ID, 10, 0)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), userAlerts, 2)

	// 3. Check symbols
	symbols := make([]string, len(userAlerts))
	for i, alert := range userAlerts {
		symbols[i] = alert.Symbol
	}
	assert.Contains(suite.T(), symbols, "BTCUSDT")
	assert.Contains(suite.T(), symbols, "ETHUSDT")
}

// Run the simple integration test suite
func TestSimpleIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SimpleIntegrationTestSuite))
}
