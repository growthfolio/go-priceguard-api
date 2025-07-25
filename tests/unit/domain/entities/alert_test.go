package entities_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestAlert_Creation(t *testing.T) {
	userID := uuid.New()
	alertID := uuid.New()

	alert := entities.Alert{
		ID:            alertID,
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
		NotifyVia:     pq.StringArray{"app", "email"},
	}

	assert.Equal(t, alertID, alert.ID)
	assert.Equal(t, userID, alert.UserID)
	assert.Equal(t, "BTCUSDT", alert.Symbol)
	assert.Equal(t, "price", alert.AlertType)
	assert.Equal(t, "above", alert.ConditionType)
	assert.Equal(t, 50000.0, alert.TargetValue)
	assert.Equal(t, "1h", alert.Timeframe)
	assert.True(t, alert.Enabled)
	assert.Contains(t, alert.NotifyVia, "app")
	assert.Contains(t, alert.NotifyVia, "email")
	assert.Nil(t, alert.TriggeredAt)
}

func TestAlert_Types(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name      string
		alertType string
		condition string
		valid     bool
	}{
		{
			name:      "price alert above",
			alertType: "price",
			condition: "above",
			valid:     true,
		},
		{
			name:      "price alert below",
			alertType: "price",
			condition: "below",
			valid:     true,
		},
		{
			name:      "rsi alert",
			alertType: "rsi",
			condition: "above",
			valid:     true,
		},
		{
			name:      "ema cross alert",
			alertType: "ema_cross",
			condition: "crosses",
			valid:     true,
		},
		{
			name:      "volume alert",
			alertType: "volume",
			condition: "above",
			valid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := entities.Alert{
				UserID:        userID,
				Symbol:        "BTCUSDT",
				AlertType:     tt.alertType,
				ConditionType: tt.condition,
				TargetValue:   100.0,
				Timeframe:     "1h",
				Enabled:       true,
			}

			assert.Equal(t, tt.alertType, alert.AlertType)
			assert.Equal(t, tt.condition, alert.ConditionType)
			assert.NotEmpty(t, alert.Symbol)
			assert.NotZero(t, alert.TargetValue)
		})
	}
}

func TestAlert_NotificationChannels(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name      string
		notifyVia pq.StringArray
		expected  []string
	}{
		{
			name:      "app only",
			notifyVia: pq.StringArray{"app"},
			expected:  []string{"app"},
		},
		{
			name:      "multiple channels",
			notifyVia: pq.StringArray{"app", "email", "sms"},
			expected:  []string{"app", "email", "sms"},
		},
		{
			name:      "empty channels",
			notifyVia: pq.StringArray{},
			expected:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := entities.Alert{
				UserID:        userID,
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   50000.0,
				Timeframe:     "1h",
				Enabled:       true,
				NotifyVia:     tt.notifyVia,
			}

			assert.Equal(t, len(tt.expected), len(alert.NotifyVia))
			for _, channel := range tt.expected {
				assert.Contains(t, alert.NotifyVia, channel)
			}
		})
	}
}

func TestAlert_Timeframes(t *testing.T) {
	userID := uuid.New()

	validTimeframes := []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w"}

	for _, timeframe := range validTimeframes {
		t.Run("timeframe_"+timeframe, func(t *testing.T) {
			alert := entities.Alert{
				UserID:        userID,
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   50000.0,
				Timeframe:     timeframe,
				Enabled:       true,
			}

			assert.Equal(t, timeframe, alert.Timeframe)
			assert.True(t, alert.Enabled)
		})
	}
}

func TestAlert_TriggerState(t *testing.T) {
	userID := uuid.New()

	alert := entities.Alert{
		UserID:        userID,
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
		Timeframe:     "1h",
		Enabled:       true,
	}

	// Initially not triggered
	assert.Nil(t, alert.TriggeredAt)

	// Simulate triggering
	now := time.Now()
	alert.TriggeredAt = &now

	assert.NotNil(t, alert.TriggeredAt)
	assert.Equal(t, now, *alert.TriggeredAt)
}

func TestAlert_EnabledState(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled alert",
			enabled: true,
		},
		{
			name:    "disabled alert",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert := entities.Alert{
				UserID:        userID,
				Symbol:        "BTCUSDT",
				AlertType:     "price",
				ConditionType: "above",
				TargetValue:   50000.0,
				Timeframe:     "1h",
				Enabled:       tt.enabled,
			}

			assert.Equal(t, tt.enabled, alert.Enabled)
		})
	}
}
