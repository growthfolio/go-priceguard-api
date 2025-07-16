package repository_test

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

type AlertRepositoryTestSuite struct {
	suite.Suite
	db   *gorm.DB
	repo repositories.AlertRepository
	ctx  context.Context
}

func (suite *AlertRepositoryTestSuite) SetupSuite() {
	// Use in-memory SQLite for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Suppress SQL logs in tests
	})
	suite.Require().NoError(err)

	// Auto-migrate the Alert and User tables
	err = db.AutoMigrate(&entities.Alert{}, &entities.User{})
	suite.Require().NoError(err)

	suite.db = db
	suite.repo = repository.NewAlertRepository(db)
	suite.ctx = context.Background()
}

func (suite *AlertRepositoryTestSuite) TearDownTest() {
	// Clean up data after each test
	suite.db.Exec("DELETE FROM alerts")
	suite.db.Exec("DELETE FROM users")
}

func (suite *AlertRepositoryTestSuite) createTestUser() *entities.User {
	user := &entities.User{
		GoogleID: "google123",
		Email:    "test@example.com",
		Name:     "Test User",
	}
	err := suite.db.Create(user).Error
	suite.Require().NoError(err)
	return user
}

func (suite *AlertRepositoryTestSuite) TestCreate_Success() {
	// Setup
	user := suite.createTestUser()
	alert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app", "email"},
	}

	// Execute
	err := suite.repo.Create(suite.ctx, alert)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, alert.ID)
	assert.False(suite.T(), alert.CreatedAt.IsZero())
	assert.False(suite.T(), alert.UpdatedAt.IsZero())
}

func (suite *AlertRepositoryTestSuite) TestGetByID_Success() {
	// Setup
	user := suite.createTestUser()
	alert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}
	err := suite.repo.Create(suite.ctx, alert)
	suite.Require().NoError(err)

	// Execute
	foundAlert, err := suite.repo.GetByID(suite.ctx, alert.ID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), foundAlert)
	assert.Equal(suite.T(), alert.ID, foundAlert.ID)
	assert.Equal(suite.T(), alert.UserID, foundAlert.UserID)
	assert.Equal(suite.T(), alert.Symbol, foundAlert.Symbol)
	assert.Equal(suite.T(), alert.AlertType, foundAlert.AlertType)
}

func (suite *AlertRepositoryTestSuite) TestGetByID_NotFound() {
	// Execute with non-existent ID
	nonExistentID := uuid.New()
	foundAlert, err := suite.repo.GetByID(suite.ctx, nonExistentID)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), foundAlert)
}

func (suite *AlertRepositoryTestSuite) TestGetByUserID_Success() {
	// Setup
	user := suite.createTestUser()

	// Create multiple alerts for the user
	alerts := []*entities.Alert{
		{
			UserID:        user.ID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
		},
		{
			UserID:        user.ID,
			Symbol:        "ETHUSDT",
			AlertType:     "price",
			ConditionType: "below",
			TargetValue:   3000.0,
			Timeframe:     "1h",
			Enabled:       true,
		},
	}

	for _, alert := range alerts {
		err := suite.repo.Create(suite.ctx, alert)
		suite.Require().NoError(err)
	}

	// Execute
	foundAlerts, err := suite.repo.GetByUserID(suite.ctx, user.ID, 10, 0)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), foundAlerts, 2)
	assert.Equal(suite.T(), user.ID, foundAlerts[0].UserID)
	assert.Equal(suite.T(), user.ID, foundAlerts[1].UserID)
}

func (suite *AlertRepositoryTestSuite) TestGetBySymbol_Success() {
	// Setup
	user := suite.createTestUser()
	symbol := "BTCUSDT"

	alert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        symbol,
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}
	err := suite.repo.Create(suite.ctx, alert)
	suite.Require().NoError(err)

	// Execute
	foundAlerts, err := suite.repo.GetBySymbol(suite.ctx, symbol)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), foundAlerts, 1)
	assert.Equal(suite.T(), symbol, foundAlerts[0].Symbol)
}

func (suite *AlertRepositoryTestSuite) TestGetEnabled_Success() {
	// Setup
	user := suite.createTestUser()

	// Create enabled and disabled alerts
	enabledAlert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}

	disabledAlert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "ETHUSDT",
		AlertType:     "price",
		ConditionType: "below",
		TargetValue:   3000.0,
		Timeframe:     "1h",
		Enabled:       false,
	}

	err := suite.repo.Create(suite.ctx, enabledAlert)
	suite.Require().NoError(err)
	err = suite.repo.Create(suite.ctx, disabledAlert)
	suite.Require().NoError(err)

	// Execute
	enabledAlerts, err := suite.repo.GetEnabled(suite.ctx)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), enabledAlerts, 1)
	assert.True(suite.T(), enabledAlerts[0].Enabled)
	assert.Equal(suite.T(), enabledAlert.ID, enabledAlerts[0].ID)
}

func (suite *AlertRepositoryTestSuite) TestUpdate_Success() {
	// Setup
	user := suite.createTestUser()
	alert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}
	err := suite.repo.Create(suite.ctx, alert)
	suite.Require().NoError(err)

	// Modify alert
	alert.TargetValue = 55000.0
	alert.Enabled = false

	// Execute
	err = suite.repo.Update(suite.ctx, alert)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify update
	foundAlert, err := suite.repo.GetByID(suite.ctx, alert.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), 55000.0, foundAlert.TargetValue)
	assert.False(suite.T(), foundAlert.Enabled)
}

func (suite *AlertRepositoryTestSuite) TestDelete_Success() {
	// Setup
	user := suite.createTestUser()
	alert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}
	err := suite.repo.Create(suite.ctx, alert)
	suite.Require().NoError(err)

	// Execute
	err = suite.repo.Delete(suite.ctx, alert.ID)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify deletion
	foundAlert, err := suite.repo.GetByID(suite.ctx, alert.ID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), foundAlert)
}

func (suite *AlertRepositoryTestSuite) TestMarkTriggered_Success() {
	// Setup
	user := suite.createTestUser()
	alert := &entities.Alert{
		UserID:        user.ID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}
	err := suite.repo.Create(suite.ctx, alert)
	suite.Require().NoError(err)

	// Execute
	err = suite.repo.MarkTriggered(suite.ctx, alert.ID)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify triggered timestamp was set
	foundAlert, err := suite.repo.GetByID(suite.ctx, alert.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), foundAlert.TriggeredAt)
	assert.False(suite.T(), foundAlert.TriggeredAt.IsZero())
}

// Run the test suite
func TestAlertRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(AlertRepositoryTestSuite))
}
