package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// AlertModelTestSuite defines basic model tests for alerts
type AlertModelTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *AlertModelTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *AlertModelTestSuite) TestAlertCreation() {
	userID := uuid.New()

	alert := &entities.Alert{
		ID:            uuid.New(),
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     pq.StringArray{"app", "email"},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Test alert structure
	assert.Equal(suite.T(), userID, alert.UserID)
	assert.Equal(suite.T(), "BTCUSDT", alert.Symbol)
	assert.Equal(suite.T(), "price", alert.AlertType)
	assert.Equal(suite.T(), "above", alert.ConditionType)
	assert.Equal(suite.T(), 50000.0, alert.TargetValue)
	assert.Equal(suite.T(), "1h", alert.Timeframe)
	assert.True(suite.T(), alert.Enabled)
	assert.Contains(suite.T(), alert.NotifyVia, "app")
	assert.Contains(suite.T(), alert.NotifyVia, "email")
	assert.Nil(suite.T(), alert.TriggeredAt)
}

func (suite *AlertModelTestSuite) TestAlertValidation() {
	tests := []struct {
		name    string
		alert   entities.Alert
		isValid bool
	}{
		{
			name: "valid price alert",
			alert: entities.Alert{
				UserID:        uuid.New(),
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   50000.0,
				Timeframe:     "1h",
				Enabled:       true,
			},
			isValid: true,
		},
		{
			name: "valid RSI alert",
			alert: entities.Alert{
				UserID:        uuid.New(),
				Symbol:        "ETHUSDT",
				AlertType:     "rsi",
				ConditionType: "below",
				TargetValue:   30.0,
				Timeframe:     "4h",
				Enabled:       true,
			},
			isValid: true,
		},
		{
			name: "valid percentage alert",
			alert: entities.Alert{
				UserID:        uuid.New(),
				Symbol:        "ADAUSDT",
				AlertType:     "percentage",
				ConditionType: "above",
				TargetValue:   5.0,
				Timeframe:     "1d",
				Enabled:       false,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			// Basic validation - ensure required fields are present
			assert.NotEqual(t, uuid.Nil, tt.alert.UserID)
			assert.NotEmpty(t, tt.alert.Symbol)
			assert.NotEmpty(t, tt.alert.AlertType)
			assert.NotEmpty(t, tt.alert.ConditionType)
			assert.NotEmpty(t, tt.alert.Timeframe)
			assert.GreaterOrEqual(t, tt.alert.TargetValue, 0.0)
		})
	}
}

func (suite *AlertModelTestSuite) TestAlertTimeframes() {
	validTimeframes := []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w"}

	for _, timeframe := range validTimeframes {
		suite.T().Run("timeframe_"+timeframe, func(t *testing.T) {
			alert := entities.Alert{
				UserID:        uuid.New(),
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   50000.0,
				Timeframe:     timeframe,
				Enabled:       true,
			}

			assert.Equal(t, timeframe, alert.Timeframe)
		})
	}
}

func (suite *AlertModelTestSuite) TestAlertTypes() {
	validTypes := []struct {
		alertType     string
		conditionType string
		targetValue   float64
	}{
		{"price", "above", 50000.0},
		{"price", "below", 45000.0},
		{"rsi", "above", 70.0},
		{"rsi", "below", 30.0},
		{"percentage", "above", 5.0},
		{"percentage", "below", -5.0},
		{"ema_cross", "up", 0.0},
		{"ema_cross", "down", 0.0},
		{"volume", "above", 1000000.0},
	}

	for _, tt := range validTypes {
		suite.T().Run(tt.alertType+"_"+tt.conditionType, func(t *testing.T) {
			alert := entities.Alert{
				UserID:        uuid.New(),
				Symbol:        "BTCUSDT",
				AlertType:     tt.alertType,
				ConditionType: tt.conditionType,
				TargetValue:   tt.targetValue,
				Timeframe:     "1h",
				Enabled:       true,
			}

			assert.Equal(t, tt.alertType, alert.AlertType)
			assert.Equal(t, tt.conditionType, alert.ConditionType)
			assert.Equal(t, tt.targetValue, alert.TargetValue)
		})
	}
}

func TestAlertModelTestSuite(t *testing.T) {
	suite.Run(t, new(AlertModelTestSuite))
}

// NotificationRepositoryTestSuite defines the test suite for notification repository
type NotificationRepositoryTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *NotificationRepositoryTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *NotificationRepositoryTestSuite) TestNotificationCreation() {
	userID := uuid.New()
	alertID := uuid.New()
	now := time.Now()

	notification := &entities.Notification{
		ID:               uuid.New(),
		UserID:           userID,
		AlertID:          &alertID,
		Title:            "Price Alert",
		Message:          "BTC reached $50,000",
		NotificationType: "alert_triggered",
		ReadAt:           nil,
		CreatedAt:        now,
	}

	// Test notification structure
	assert.Equal(suite.T(), userID, notification.UserID)
	assert.Equal(suite.T(), &alertID, notification.AlertID)
	assert.Equal(suite.T(), "Price Alert", notification.Title)
	assert.Equal(suite.T(), "BTC reached $50,000", notification.Message)
	assert.Equal(suite.T(), "alert_triggered", notification.NotificationType)
	assert.Nil(suite.T(), notification.ReadAt)
	assert.Equal(suite.T(), now, notification.CreatedAt)
}

func (suite *NotificationRepositoryTestSuite) TestNotificationTypes() {
	userID := uuid.New()

	tests := []struct {
		name             string
		notificationType string
		hasAlert         bool
	}{
		{
			name:             "alert triggered notification",
			notificationType: "alert_triggered",
			hasAlert:         true,
		},
		{
			name:             "system notification",
			notificationType: "system",
			hasAlert:         false,
		},
		{
			name:             "account notification",
			notificationType: "account",
			hasAlert:         false,
		},
		{
			name:             "market update notification",
			notificationType: "market_update",
			hasAlert:         false,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			var alertID *uuid.UUID
			if tt.hasAlert {
				id := uuid.New()
				alertID = &id
			}

			notification := entities.Notification{
				UserID:           userID,
				AlertID:          alertID,
				Title:            "Test Title",
				Message:          "Test Message",
				NotificationType: tt.notificationType,
			}

			assert.Equal(t, tt.notificationType, notification.NotificationType)
			if tt.hasAlert {
				assert.NotNil(t, notification.AlertID)
			} else {
				assert.Nil(t, notification.AlertID)
			}
		})
	}
}

func (suite *NotificationRepositoryTestSuite) TestNotificationReadState() {
	userID := uuid.New()

	tests := []struct {
		name   string
		readAt *time.Time
		isRead bool
	}{
		{
			name:   "unread notification",
			readAt: nil,
			isRead: false,
		},
		{
			name:   "read notification",
			readAt: func() *time.Time { t := time.Now(); return &t }(),
			isRead: true,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			notification := entities.Notification{
				UserID:           userID,
				Title:            "Test",
				Message:          "Test",
				NotificationType: "system",
				ReadAt:           tt.readAt,
			}

			isRead := notification.ReadAt != nil
			assert.Equal(t, tt.isRead, isRead)
		})
	}
}

func TestNotificationRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(NotificationRepositoryTestSuite))
}

// PriceHistoryRepositoryTestSuite defines the test suite for price history repository
type PriceHistoryRepositoryTestSuite struct {
	suite.Suite
	ctx context.Context
}

func (suite *PriceHistoryRepositoryTestSuite) SetupTest() {
	suite.ctx = context.Background()
}

func (suite *PriceHistoryRepositoryTestSuite) TestPriceHistoryCreation() {
	now := time.Now()

	priceHistory := &entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		OpenPrice:  50000.0,
		HighPrice:  52000.0,
		LowPrice:   49000.0,
		ClosePrice: 51000.0,
		Volume:     1000.0,
		Timestamp:  now,
	}

	// Test price history structure
	assert.Equal(suite.T(), "BTCUSDT", priceHistory.Symbol)
	assert.Equal(suite.T(), "1h", priceHistory.Timeframe)
	assert.Equal(suite.T(), 50000.0, priceHistory.OpenPrice)
	assert.Equal(suite.T(), 52000.0, priceHistory.HighPrice)
	assert.Equal(suite.T(), 49000.0, priceHistory.LowPrice)
	assert.Equal(suite.T(), 51000.0, priceHistory.ClosePrice)
	assert.Equal(suite.T(), 1000.0, priceHistory.Volume)
	assert.Equal(suite.T(), now, priceHistory.Timestamp)
}

func (suite *PriceHistoryRepositoryTestSuite) TestOHLCValidation() {
	tests := []struct {
		name       string
		openPrice  float64
		highPrice  float64
		lowPrice   float64
		closePrice float64
		isValid    bool
	}{
		{
			name:       "valid OHLC",
			openPrice:  50000.0,
			highPrice:  52000.0,
			lowPrice:   49000.0,
			closePrice: 51000.0,
			isValid:    true,
		},
		{
			name:       "high equals all prices",
			openPrice:  50000.0,
			highPrice:  50000.0,
			lowPrice:   50000.0,
			closePrice: 50000.0,
			isValid:    true,
		},
	}

	for _, tt := range tests {
		suite.T().Run(tt.name, func(t *testing.T) {
			priceHistory := entities.PriceHistory{
				Symbol:     "BTCUSDT",
				Timeframe:  "1h",
				OpenPrice:  tt.openPrice,
				HighPrice:  tt.highPrice,
				LowPrice:   tt.lowPrice,
				ClosePrice: tt.closePrice,
				Volume:     1000.0,
				Timestamp:  time.Now(),
			}

			// Validate OHLC relationships
			assert.GreaterOrEqual(t, priceHistory.HighPrice, priceHistory.OpenPrice)
			assert.GreaterOrEqual(t, priceHistory.HighPrice, priceHistory.ClosePrice)
			assert.LessOrEqual(t, priceHistory.LowPrice, priceHistory.OpenPrice)
			assert.LessOrEqual(t, priceHistory.LowPrice, priceHistory.ClosePrice)
		})
	}
}

func TestPriceHistoryRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PriceHistoryRepositoryTestSuite))
}
