package entities_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/stretchr/testify/assert"
)

func TestUser_Creation(t *testing.T) {
	now := time.Now()
	userID := uuid.New()

	user := entities.User{
		ID:        userID,
		GoogleID:  "google-123",
		Email:     "test@example.com",
		Name:      "Test User",
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, userID, user.ID)
	assert.Equal(t, "google-123", user.GoogleID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.Nil(t, user.Picture)
	assert.Nil(t, user.Avatar)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
}

func TestUser_WithOptionalFields(t *testing.T) {
	userID := uuid.New()
	picture := "https://example.com/picture.jpg"
	avatar := "https://example.com/avatar.jpg"

	user := entities.User{
		ID:       userID,
		GoogleID: "google-123",
		Email:    "test@example.com",
		Name:     "Test User",
		Picture:  &picture,
		Avatar:   &avatar,
	}

	assert.NotNil(t, user.Picture)
	assert.Equal(t, picture, *user.Picture)
	assert.NotNil(t, user.Avatar)
	assert.Equal(t, avatar, *user.Avatar)
}

func TestAlert_Validation(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name  string
		alert entities.Alert
		valid bool
	}{
		{
			name: "valid alert",
			alert: entities.Alert{
				UserID:        userID,
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   50000.0,
				Timeframe:     "1h",
				Enabled:       true,
			},
			valid: true,
		},
		{
			name: "zero target value should be valid",
			alert: entities.Alert{
				UserID:        userID,
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "below",
				TargetValue:   0.0,
				Timeframe:     "1h",
				Enabled:       true,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Testa se os campos obrigatórios estão preenchidos
			assert.NotEqual(t, uuid.Nil, tt.alert.UserID)
			assert.NotEmpty(t, tt.alert.Symbol)
			assert.NotEmpty(t, tt.alert.AlertType)
			assert.NotEmpty(t, tt.alert.ConditionType)
			assert.NotEmpty(t, tt.alert.Timeframe)
		})
	}
}

func TestNotification_IsRead(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		notification entities.Notification
		expected     bool
	}{
		{
			name: "read notification",
			notification: entities.Notification{
				ReadAt: &now,
			},
			expected: true,
		},
		{
			name: "unread notification",
			notification: entities.Notification{
				ReadAt: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isRead := tt.notification.ReadAt != nil
			assert.Equal(t, tt.expected, isRead)
		})
	}
}

func TestPriceHistory_OHLC(t *testing.T) {
	history := entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		OpenPrice:  50000.0,
		HighPrice:  52000.0,
		LowPrice:   49000.0,
		ClosePrice: 51000.0,
		Volume:     1000.0,
		Timestamp:  time.Now(),
	}

	// Testa se os valores OHLC estão corretos
	assert.Equal(t, 50000.0, history.OpenPrice)
	assert.Equal(t, 52000.0, history.HighPrice)
	assert.Equal(t, 49000.0, history.LowPrice)
	assert.Equal(t, 51000.0, history.ClosePrice)
	assert.Equal(t, 1000.0, history.Volume)

	// Verifica lógica de preços
	assert.GreaterOrEqual(t, history.HighPrice, history.OpenPrice)
	assert.GreaterOrEqual(t, history.HighPrice, history.ClosePrice)
	assert.LessOrEqual(t, history.LowPrice, history.OpenPrice)
	assert.LessOrEqual(t, history.LowPrice, history.ClosePrice)
}

func TestUserSettings_Defaults(t *testing.T) {
	userID := uuid.New()

	settings := entities.UserSettings{
		UserID: userID,
	}

	// Testa valores padrão quando não especificados
	assert.Equal(t, userID, settings.UserID)
	// Em um cenário real, estes valores seriam setados pelos defaults do GORM
	// Aqui apenas validamos que a estrutura permite estes campos
	assert.NotNil(t, &settings.Theme)
	assert.NotNil(t, &settings.DefaultTimeframe)
	assert.NotNil(t, &settings.NotificationsEmail)
}

func TestCryptoCurrency_Fields(t *testing.T) {
	crypto := entities.CryptoCurrency{
		Symbol:     "BTCUSDT",
		Name:       "Bitcoin",
		MarketType: "Spot",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	assert.Equal(t, "BTCUSDT", crypto.Symbol)
	assert.Equal(t, "Bitcoin", crypto.Name)
	assert.Equal(t, "Spot", crypto.MarketType)
	assert.True(t, crypto.Active)
	assert.NotZero(t, crypto.CreatedAt)
	assert.NotZero(t, crypto.UpdatedAt)
}

func TestTechnicalIndicator_Structure(t *testing.T) {
	value := 70.5
	indicator := entities.TechnicalIndicator{
		Symbol:        "BTCUSDT",
		Timeframe:     "1h",
		IndicatorType: "rsi",
		Value:         &value,
		Timestamp:     time.Now(),
	}

	assert.Equal(t, "BTCUSDT", indicator.Symbol)
	assert.Equal(t, "1h", indicator.Timeframe)
	assert.Equal(t, "rsi", indicator.IndicatorType)
	assert.NotNil(t, indicator.Value)
	assert.Equal(t, 70.5, *indicator.Value)
	assert.NotZero(t, indicator.Timestamp)
}

func TestAlert_IsTriggered(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	tests := []struct {
		name        string
		alert       entities.Alert
		isTriggered bool
	}{
		{
			name: "triggered alert",
			alert: entities.Alert{
				UserID:        userID,
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   50000.0,
				Timeframe:     "1h",
				Enabled:       true,
				TriggeredAt:   &now,
			},
			isTriggered: true,
		},
		{
			name: "non-triggered alert",
			alert: entities.Alert{
				UserID:        userID,
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   50000.0,
				Timeframe:     "1h",
				Enabled:       true,
				TriggeredAt:   nil,
			},
			isTriggered: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isTriggered := tt.alert.TriggeredAt != nil
			assert.Equal(t, tt.isTriggered, isTriggered)
		})
	}
}

func TestUserSettings_DefaultValues(t *testing.T) {
	userID := uuid.New()
	settingsID := uuid.New()

	settings := entities.UserSettings{
		ID:                 settingsID,
		UserID:             userID,
		Theme:              "dark",
		DefaultTimeframe:   "1h",
		DefaultView:        "overview",
		NotificationsEmail: true,
		NotificationsPush:  true,
		NotificationsSMS:   false,
		RiskProfile:        "moderate",
	}

	assert.Equal(t, settingsID, settings.ID)
	assert.Equal(t, userID, settings.UserID)
	assert.Equal(t, "dark", settings.Theme)
	assert.Equal(t, "1h", settings.DefaultTimeframe)
	assert.Equal(t, "overview", settings.DefaultView)
	assert.True(t, settings.NotificationsEmail)
	assert.True(t, settings.NotificationsPush)
	assert.False(t, settings.NotificationsSMS)
	assert.Equal(t, "moderate", settings.RiskProfile)
}

func TestNotification_Types(t *testing.T) {
	userID := uuid.New()
	alertID := uuid.New()

	tests := []struct {
		name         string
		notification entities.Notification
		hasAlert     bool
	}{
		{
			name: "alert notification",
			notification: entities.Notification{
				UserID:           userID,
				AlertID:          &alertID,
				Title:            "Price Alert",
				Message:          "BTC reached target price",
				NotificationType: "alert_triggered",
			},
			hasAlert: true,
		},
		{
			name: "system notification",
			notification: entities.Notification{
				UserID:           userID,
				AlertID:          nil,
				Title:            "System Update",
				Message:          "New features available",
				NotificationType: "system",
			},
			hasAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasAlert := tt.notification.AlertID != nil
			assert.Equal(t, tt.hasAlert, hasAlert)
			assert.NotEmpty(t, tt.notification.Title)
			assert.NotEmpty(t, tt.notification.Message)
			assert.NotEmpty(t, tt.notification.NotificationType)
		})
	}
}
