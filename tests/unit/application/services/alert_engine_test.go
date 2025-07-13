package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/tests/testutils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AlertEngineTestSuite struct {
	suite.Suite
	alertEngine                *services.AlertEngine
	mockAlertRepo              *testutils.MockAlertRepository
	mockPriceHistoryRepo       *testutils.MockPriceHistoryRepository
	mockTechnicalIndicatorRepo *testutils.MockTechnicalIndicatorRepository
	mockNotificationRepo       *testutils.MockNotificationRepository
	logger                     *logrus.Logger
	ctx                        context.Context
}

func (suite *AlertEngineTestSuite) SetupTest() {
	suite.mockAlertRepo = &testutils.MockAlertRepository{}
	suite.mockPriceHistoryRepo = &testutils.MockPriceHistoryRepository{}
	suite.mockTechnicalIndicatorRepo = &testutils.MockTechnicalIndicatorRepository{}
	suite.mockNotificationRepo = &testutils.MockNotificationRepository{}
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	suite.ctx = context.Background()

	suite.alertEngine = services.NewAlertEngine(
		suite.mockAlertRepo,
		suite.mockPriceHistoryRepo,
		suite.mockTechnicalIndicatorRepo,
		suite.mockNotificationRepo,
		nil, // TechnicalIndicatorService will be mocked if needed
		suite.logger,
	)
}

func (suite *AlertEngineTestSuite) TearDownTest() {
	suite.mockAlertRepo.AssertExpectations(suite.T())
	suite.mockPriceHistoryRepo.AssertExpectations(suite.T())
	suite.mockTechnicalIndicatorRepo.AssertExpectations(suite.T())
	suite.mockNotificationRepo.AssertExpectations(suite.T())
}

func (suite *AlertEngineTestSuite) TestNewAlertEngine() {
	// Test that NewAlertEngine creates a valid instance
	assert.NotNil(suite.T(), suite.alertEngine)
}

func (suite *AlertEngineTestSuite) TestEvaluateAllAlerts_NoAlerts() {
	// Setup: No alerts in the system
	suite.mockAlertRepo.On("GetEnabled", suite.ctx).Return([]entities.Alert{}, nil)

	// Execute
	results, err := suite.alertEngine.EvaluateAllAlerts(suite.ctx)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), results)
}

func (suite *AlertEngineTestSuite) TestEvaluateAllAlerts_ErrorGettingAlerts() {
	// Setup: Error when getting alerts
	expectedError := assert.AnError
	suite.mockAlertRepo.On("GetEnabled", suite.ctx).Return([]entities.Alert{}, expectedError)

	// Execute
	results, err := suite.alertEngine.EvaluateAllAlerts(suite.ctx)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), results)
	assert.Contains(suite.T(), err.Error(), "failed to get enabled alerts")
}

func (suite *AlertEngineTestSuite) TestEvaluateAllAlerts_WithAlerts() {
	// Setup: Create test alerts
	alertID1 := uuid.New()
	alertID2 := uuid.New()
	userID := uuid.New()

	alerts := []entities.Alert{
		{
			ID:            alertID1,
			UserID:        userID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            alertID2,
			UserID:        userID,
			Symbol:        "ETHUSDT",
			AlertType:     "price",
			ConditionType: "below",
			TargetValue:   3000.0,
			Timeframe:     "1h",
			Enabled:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	suite.mockAlertRepo.On("GetEnabled", suite.ctx).Return(alerts, nil)

	// Mock price data for evaluation - using ClosePrice field
	btcPriceHistory := []entities.PriceHistory{
		{
			Symbol:     "BTCUSDT",
			Timeframe:  "1h",
			OpenPrice:  50500.0,
			HighPrice:  51200.0,
			LowPrice:   50400.0,
			ClosePrice: 51000.0,
			Volume:     1000.0,
			Timestamp:  time.Now(),
		},
	}
	ethPriceHistory := []entities.PriceHistory{
		{
			Symbol:     "ETHUSDT",
			Timeframe:  "1h",
			OpenPrice:  2950.0,
			HighPrice:  2980.0,
			LowPrice:   2890.0,
			ClosePrice: 2900.0,
			Volume:     1500.0,
			Timestamp:  time.Now(),
		},
	}

	suite.mockPriceHistoryRepo.On("GetLatest", suite.ctx, "BTCUSDT", 1).Return(btcPriceHistory, nil)
	suite.mockPriceHistoryRepo.On("GetLatest", suite.ctx, "ETHUSDT", 1).Return(ethPriceHistory, nil)

	// Mock notification creation for triggered alerts
	suite.mockNotificationRepo.On("Create", suite.ctx, mock.AnythingOfType("*entities.Notification")).Return(nil).Times(2)

	// Execute
	results, err := suite.alertEngine.EvaluateAllAlerts(suite.ctx)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), results, 2)

	// Check first alert (BTC price above 50000 - should trigger)
	btcResult := findResultByAlertID(results, alertID1)
	assert.NotNil(suite.T(), btcResult)
	assert.True(suite.T(), btcResult.ShouldTrigger)
	assert.Equal(suite.T(), 51000.0, btcResult.CurrentValue)
	assert.Equal(suite.T(), 50000.0, btcResult.TargetValue)

	// Check second alert (ETH price below 3000 - should trigger)
	ethResult := findResultByAlertID(results, alertID2)
	assert.NotNil(suite.T(), ethResult)
	assert.True(suite.T(), ethResult.ShouldTrigger)
	assert.Equal(suite.T(), 2900.0, ethResult.CurrentValue)
	assert.Equal(suite.T(), 3000.0, ethResult.TargetValue)
}

// Helper function to find a result by alert ID
func findResultByAlertID(results []services.AlertEvaluationResult, alertID uuid.UUID) *services.AlertEvaluationResult {
	for i := range results {
		if results[i].AlertID == alertID {
			return &results[i]
		}
	}
	return nil
}

func TestAlertEngineTestSuite(t *testing.T) {
	suite.Run(t, new(AlertEngineTestSuite))
}

// Additional unit tests for specific methods
func TestAlertEngine_EvaluateAlert_PriceConditions(t *testing.T) {
	mockAlertRepo := &testutils.MockAlertRepository{}
	mockPriceHistoryRepo := &testutils.MockPriceHistoryRepository{}
	mockTechnicalIndicatorRepo := &testutils.MockTechnicalIndicatorRepository{}
	mockNotificationRepo := &testutils.MockNotificationRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ctx := context.Background()

	alertEngine := services.NewAlertEngine(
		mockAlertRepo,
		mockPriceHistoryRepo,
		mockTechnicalIndicatorRepo,
		mockNotificationRepo,
		nil,
		logger,
	)

	t.Run("price_above_should_trigger", func(t *testing.T) {
		alertID := uuid.New()
		userID := uuid.New()
		alert := &entities.Alert{
			ID:            alertID,
			UserID:        userID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		priceHistory := []entities.PriceHistory{
			{
				Symbol:     "BTCUSDT",
				Timeframe:  "1h",
				OpenPrice:  50500.0,
				HighPrice:  51200.0,
				LowPrice:   50400.0,
				ClosePrice: 51000.0,
				Volume:     1000.0,
				Timestamp:  time.Now(),
			},
		}

		mockPriceHistoryRepo.On("GetLatest", ctx, "BTCUSDT", 1).Return(priceHistory, nil)
		mockNotificationRepo.On("Create", ctx, mock.AnythingOfType("*entities.Notification")).Return(nil)

		result, err := alertEngine.EvaluateAlert(ctx, alert)

		assert.NoError(t, err)
		assert.True(t, result.ShouldTrigger)
		assert.Equal(t, 51000.0, result.CurrentValue)
		assert.Equal(t, 50000.0, result.TargetValue)
		assert.Contains(t, result.Message, "Price Alert")

		mockPriceHistoryRepo.AssertExpectations(t)
		mockNotificationRepo.AssertExpectations(t)
	})

	t.Run("price_above_should_not_trigger", func(t *testing.T) {
		alertID := uuid.New()
		userID := uuid.New()
		alert := &entities.Alert{
			ID:            alertID,
			UserID:        userID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		priceHistory := []entities.PriceHistory{
			{
				Symbol:     "BTCUSDT",
				Timeframe:  "1h",
				OpenPrice:  48500.0,
				HighPrice:  49200.0,
				LowPrice:   48400.0,
				ClosePrice: 49000.0,
				Volume:     1000.0,
				Timestamp:  time.Now(),
			},
		}

		mockPriceHistoryRepo.On("GetLatest", ctx, "BTCUSDT", 1).Return(priceHistory, nil)

		result, err := alertEngine.EvaluateAlert(ctx, alert)

		assert.NoError(t, err)
		assert.False(t, result.ShouldTrigger)
		assert.Equal(t, 49000.0, result.CurrentValue)
		assert.Equal(t, 50000.0, result.TargetValue)

		mockPriceHistoryRepo.AssertExpectations(t)
	})

	t.Run("price_below_should_trigger", func(t *testing.T) {
		alertID := uuid.New()
		userID := uuid.New()
		alert := &entities.Alert{
			ID:            alertID,
			UserID:        userID,
			Symbol:        "ETHUSDT",
			AlertType:     "price",
			ConditionType: "below",
			TargetValue:   3000.0,
			Timeframe:     "1h",
			Enabled:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		priceHistory := []entities.PriceHistory{
			{
				Symbol:     "ETHUSDT",
				Timeframe:  "1h",
				OpenPrice:  2950.0,
				HighPrice:  2980.0,
				LowPrice:   2890.0,
				ClosePrice: 2900.0,
				Volume:     1500.0,
				Timestamp:  time.Now(),
			},
		}

		mockPriceHistoryRepo.On("GetLatest", ctx, "ETHUSDT", 1).Return(priceHistory, nil)
		mockNotificationRepo.On("Create", ctx, mock.AnythingOfType("*entities.Notification")).Return(nil)

		result, err := alertEngine.EvaluateAlert(ctx, alert)

		assert.NoError(t, err)
		assert.True(t, result.ShouldTrigger)
		assert.Equal(t, 2900.0, result.CurrentValue)
		assert.Equal(t, 3000.0, result.TargetValue)

		mockPriceHistoryRepo.AssertExpectations(t)
		mockNotificationRepo.AssertExpectations(t)
	})

	t.Run("error_getting_price_data", func(t *testing.T) {
		alertID := uuid.New()
		userID := uuid.New()
		alert := &entities.Alert{
			ID:            alertID,
			UserID:        userID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		expectedError := assert.AnError
		mockPriceHistoryRepo.On("GetLatest", ctx, "BTCUSDT", 1).Return([]entities.PriceHistory{}, expectedError)

		result, err := alertEngine.EvaluateAlert(ctx, alert)

		assert.Error(t, err)
		assert.Equal(t, services.AlertEvaluationResult{}, result)
		assert.Contains(t, err.Error(), "failed to get latest price")
	})
}

func TestAlertEngine_CleanupThrottles(t *testing.T) {
	mockAlertRepo := &testutils.MockAlertRepository{}
	mockPriceHistoryRepo := &testutils.MockPriceHistoryRepository{}
	mockTechnicalIndicatorRepo := &testutils.MockTechnicalIndicatorRepository{}
	mockNotificationRepo := &testutils.MockNotificationRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	alertEngine := services.NewAlertEngine(
		mockAlertRepo,
		mockPriceHistoryRepo,
		mockTechnicalIndicatorRepo,
		mockNotificationRepo,
		nil,
		logger,
	)

	t.Run("cleanup_throttles_executes_without_error", func(t *testing.T) {
		// Test that CleanupThrottles can be called without error
		assert.NotPanics(t, func() {
			alertEngine.CleanupThrottles()
		})
	})
}

func TestAlertEngine_GetAlertStats(t *testing.T) {
	mockAlertRepo := &testutils.MockAlertRepository{}
	mockPriceHistoryRepo := &testutils.MockPriceHistoryRepository{}
	mockTechnicalIndicatorRepo := &testutils.MockTechnicalIndicatorRepository{}
	mockNotificationRepo := &testutils.MockNotificationRepository{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	ctx := context.Background()

	alertEngine := services.NewAlertEngine(
		mockAlertRepo,
		mockPriceHistoryRepo,
		mockTechnicalIndicatorRepo,
		mockNotificationRepo,
		nil,
		logger,
	)

	t.Run("get_alert_stats_success", func(t *testing.T) {
		// Mock enabled alerts
		mockAlertRepo.On("GetEnabled", ctx).Return([]entities.Alert{
			{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				Symbol:    "BTCUSDT",
				AlertType: "price",
				Enabled:   true,
			},
		}, nil)

		stats, err := alertEngine.GetAlertStats(ctx)

		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "total_enabled_alerts")

		mockAlertRepo.AssertExpectations(t)
	})

	t.Run("get_alert_stats_error", func(t *testing.T) {
		expectedError := assert.AnError
		mockAlertRepo.On("GetEnabled", ctx).Return([]entities.Alert{}, expectedError)

		stats, err := alertEngine.GetAlertStats(ctx)

		assert.Error(t, err)
		assert.Nil(t, stats)

		mockAlertRepo.AssertExpectations(t)
	})
}

func (suite *AlertEngineTestSuite) TestSetWebSocketService() {
	// Setup
	mockWebSocketService := &testutils.MockAlertWebSocketService{}

	// Execute
	suite.alertEngine.SetWebSocketService(mockWebSocketService)

	// Assert - no direct way to test this since it's a setter, but should not panic
	assert.NotNil(suite.T(), suite.alertEngine)
}

func (suite *AlertEngineTestSuite) TestEvaluateAlert_PriceAbove_Triggered() {
	// Setup
	alertID := uuid.New()
	userID := uuid.New()
	alert := &entities.Alert{
		ID:            alertID,
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}

	// Mock price data that triggers the alert
	priceData := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		ClosePrice: 55000.0, // Above target
		Timestamp:  time.Now(),
	}

	suite.mockPriceHistoryRepo.On("GetLatest", suite.ctx, "BTCUSDT", "1h").Return(priceData, nil)

	// Execute
	result, err := suite.alertEngine.EvaluateAlert(suite.ctx, alert)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.ShouldTrigger)
	assert.Equal(suite.T(), 55000.0, result.CurrentValue)
	assert.Equal(suite.T(), 50000.0, result.TargetValue)
}

func (suite *AlertEngineTestSuite) TestEvaluateAlert_PriceBelow_NotTriggered() {
	// Setup
	alertID := uuid.New()
	userID := uuid.New()
	alert := &entities.Alert{
		ID:            alertID,
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "below",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}

	// Mock price data that doesn't trigger the alert
	priceData := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		ClosePrice: 55000.0, // Above target, should not trigger "below"
		Timestamp:  time.Now(),
	}

	suite.mockPriceHistoryRepo.On("GetLatest", suite.ctx, "BTCUSDT", "1h").Return(priceData, nil)

	// Execute
	result, err := suite.alertEngine.EvaluateAlert(suite.ctx, alert)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.False(suite.T(), result.ShouldTrigger)
	assert.Equal(suite.T(), 55000.0, result.CurrentValue)
	assert.Equal(suite.T(), 50000.0, result.TargetValue)
}

func (suite *AlertEngineTestSuite) TestEvaluateAlert_ErrorGettingPriceData() {
	// Setup
	alertID := uuid.New()
	userID := uuid.New()
	alert := &entities.Alert{
		ID:            alertID,
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}

	expectedError := assert.AnError
	suite.mockPriceHistoryRepo.On("GetLatest", suite.ctx, "BTCUSDT", "1h").Return(nil, expectedError)

	// Execute
	result, err := suite.alertEngine.EvaluateAlert(suite.ctx, alert)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "failed to get latest price data")
}

func (suite *AlertEngineTestSuite) TestEvaluateAlert_PercentageUp_Triggered() {
	// Setup
	alertID := uuid.New()
	userID := uuid.New()
	alert := &entities.Alert{
		ID:            alertID,
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "percentage",
		ConditionType: "up",
		TargetValue:   5.0, // 5% increase
		Timeframe:     "1h",
		Enabled:       true,
	}

	// Current price data
	currentPrice := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		ClosePrice: 52500.0, // 5% increase from 50000
		Timestamp:  time.Now(),
	}

	// Previous price data (1 hour ago)
	previousPrice := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		ClosePrice: 50000.0,
		Timestamp:  time.Now().Add(-time.Hour),
	}

	priceHistory := []entities.PriceHistory{*previousPrice, *currentPrice}

	suite.mockPriceHistoryRepo.On("GetLatest", suite.ctx, "BTCUSDT", "1h").Return(currentPrice, nil)
	suite.mockPriceHistoryRepo.On("GetBySymbol", suite.ctx, "BTCUSDT", "1h", 2).Return(priceHistory, nil)

	// Execute
	result, err := suite.alertEngine.EvaluateAlert(suite.ctx, alert)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.ShouldTrigger)
}

func (suite *AlertEngineTestSuite) TestEvaluateAlert_UnsupportedAlertType() {
	// Setup
	alertID := uuid.New()
	userID := uuid.New()
	alert := &entities.Alert{
		ID:            alertID,
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "unsupported",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}

	priceData := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		ClosePrice: 55000.0,
		Timestamp:  time.Now(),
	}

	suite.mockPriceHistoryRepo.On("GetLatest", suite.ctx, "BTCUSDT", "1h").Return(priceData, nil)

	// Execute
	result, err := suite.alertEngine.EvaluateAlert(suite.ctx, alert)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Contains(suite.T(), err.Error(), "unsupported alert type")
}

func (suite *AlertEngineTestSuite) TestProcessTriggeredAlert_Success() {
	// This is a private method, so we test it indirectly through EvaluateAlert
	// by setting up a complete evaluation scenario
	alertID := uuid.New()
	userID := uuid.New()
	alert := &entities.Alert{
		ID:            alertID,
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     []string{"app", "email"},
	}

	priceData := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		ClosePrice: 55000.0,
		Timestamp:  time.Now(),
	}

	suite.mockPriceHistoryRepo.On("GetLatest", suite.ctx, "BTCUSDT", "1h").Return(priceData, nil)
	suite.mockAlertRepo.On("MarkTriggered", suite.ctx, alertID).Return(nil)
	suite.mockNotificationRepo.On("Create", suite.ctx, mock.MatchedBy(func(n *entities.Notification) bool {
		return n.UserID == userID && n.AlertID != nil && *n.AlertID == alertID
	})).Return(nil)

	// Execute
	result, err := suite.alertEngine.EvaluateAlert(suite.ctx, alert)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	assert.True(suite.T(), result.ShouldTrigger)
}

func (suite *AlertEngineTestSuite) TestGetAlertStats_Success() {
	// Mock the repository to return some test data
	alerts := []entities.Alert{
		{ID: uuid.New(), Enabled: true},
		{ID: uuid.New(), Enabled: true},
		{ID: uuid.New(), Enabled: false},
	}

	suite.mockAlertRepo.On("GetEnabled", suite.ctx).Return(alerts[:2], nil)

	// Execute
	stats, err := suite.alertEngine.GetAlertStats(suite.ctx)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), stats)
	// Note: The actual implementation may return different stats,
	// this is just testing that the method doesn't error
}

func (suite *AlertEngineTestSuite) TestGetAlertStats_Error() {
	// Setup
	expectedError := assert.AnError
	suite.mockAlertRepo.On("GetEnabled", suite.ctx).Return(nil, expectedError)

	// Execute
	stats, err := suite.alertEngine.GetAlertStats(suite.ctx)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), stats)
}

// Test concurrent evaluation safety
func (suite *AlertEngineTestSuite) TestConcurrentEvaluation() {
	// Setup multiple alerts
	alerts := make([]entities.Alert, 10)
	for i := range alerts {
		alerts[i] = entities.Alert{
			ID:            uuid.New(),
			UserID:        uuid.New(),
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Timeframe:     "1h",
			Enabled:       true,
		}
	}

	priceData := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		ClosePrice: 55000.0,
		Timestamp:  time.Now(),
	}

	// Setup mocks for concurrent access
	suite.mockAlertRepo.On("GetEnabled", suite.ctx).Return(alerts, nil)
	for range alerts {
		suite.mockPriceHistoryRepo.On("GetLatest", suite.ctx, "BTCUSDT", "1h").Return(priceData, nil)
		suite.mockAlertRepo.On("MarkTriggered", suite.ctx, mock.AnythingOfType("uuid.UUID")).Return(nil)
		suite.mockNotificationRepo.On("Create", suite.ctx, mock.AnythingOfType("*entities.Notification")).Return(nil)
	}

	// Execute
	results, err := suite.alertEngine.EvaluateAllAlerts(suite.ctx)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), results, len(alerts))
}
