package entities_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/stretchr/testify/assert"
)

func TestNotification_Creation(t *testing.T) {
	userID := uuid.New()
	alertID := uuid.New()
	notificationID := uuid.New()
	now := time.Now()

	notification := entities.Notification{
		ID:               notificationID,
		UserID:           userID,
		AlertID:          &alertID,
		Title:            "Price Alert Triggered",
		Message:          "BTC has reached your target price of $50,000",
		NotificationType: "alert_triggered",
		CreatedAt:        now,
	}

	assert.Equal(t, notificationID, notification.ID)
	assert.Equal(t, userID, notification.UserID)
	assert.NotNil(t, notification.AlertID)
	assert.Equal(t, alertID, *notification.AlertID)
	assert.Equal(t, "Price Alert Triggered", notification.Title)
	assert.Equal(t, "BTC has reached your target price of $50,000", notification.Message)
	assert.Equal(t, "alert_triggered", notification.NotificationType)
	assert.Equal(t, now, notification.CreatedAt)
	assert.Nil(t, notification.ReadAt)
}

func TestNotification_TypesDetailed(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name             string
		notificationType string
		hasAlert         bool
		alertID          *uuid.UUID
	}{
		{
			name:             "alert triggered notification",
			notificationType: "alert_triggered",
			hasAlert:         true,
			alertID:          func() *uuid.UUID { id := uuid.New(); return &id }(),
		},
		{
			name:             "system notification",
			notificationType: "system",
			hasAlert:         false,
			alertID:          nil,
		},
		{
			name:             "account notification",
			notificationType: "account",
			hasAlert:         false,
			alertID:          nil,
		},
		{
			name:             "market update notification",
			notificationType: "market_update",
			hasAlert:         false,
			alertID:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notification := entities.Notification{
				UserID:           userID,
				AlertID:          tt.alertID,
				Title:            "Test Notification",
				Message:          "Test message",
				NotificationType: tt.notificationType,
			}

			assert.Equal(t, tt.notificationType, notification.NotificationType)
			assert.Equal(t, tt.hasAlert, notification.AlertID != nil)
			if tt.hasAlert {
				assert.Equal(t, *tt.alertID, *notification.AlertID)
			}
		})
	}
}

func TestNotification_ReadState(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			notification := entities.Notification{
				UserID:           userID,
				Title:            "Test Notification",
				Message:          "Test message",
				NotificationType: "system",
				ReadAt:           tt.readAt,
			}

			isRead := notification.ReadAt != nil
			assert.Equal(t, tt.isRead, isRead)

			if tt.isRead {
				assert.NotNil(t, notification.ReadAt)
			} else {
				assert.Nil(t, notification.ReadAt)
			}
		})
	}
}

func TestNotification_MarkAsRead(t *testing.T) {
	userID := uuid.New()

	notification := entities.Notification{
		UserID:           userID,
		Title:            "Test Notification",
		Message:          "Test message",
		NotificationType: "system",
		ReadAt:           nil, // Initially unread
	}

	// Initially unread
	assert.Nil(t, notification.ReadAt)

	// Mark as read
	now := time.Now()
	notification.ReadAt = &now

	// Now it's read
	assert.NotNil(t, notification.ReadAt)
	assert.Equal(t, now, *notification.ReadAt)
}

func TestNotification_RequiredFields(t *testing.T) {
	userID := uuid.New()

	notification := entities.Notification{
		UserID:           userID,
		Title:            "Required Title",
		Message:          "Required Message",
		NotificationType: "system",
	}

	// Test that required fields are not empty
	assert.NotEqual(t, uuid.Nil, notification.UserID)
	assert.NotEmpty(t, notification.Title)
	assert.NotEmpty(t, notification.Message)
	assert.NotEmpty(t, notification.NotificationType)
}

func TestNotification_WithOptionalAlert(t *testing.T) {
	userID := uuid.New()
	alertID := uuid.New()

	// Notification with alert
	notificationWithAlert := entities.Notification{
		UserID:           userID,
		AlertID:          &alertID,
		Title:            "Alert Notification",
		Message:          "Your alert was triggered",
		NotificationType: "alert_triggered",
	}

	// Notification without alert
	notificationWithoutAlert := entities.Notification{
		UserID:           userID,
		AlertID:          nil,
		Title:            "System Notification",
		Message:          "System update available",
		NotificationType: "system",
	}

	assert.NotNil(t, notificationWithAlert.AlertID)
	assert.Equal(t, alertID, *notificationWithAlert.AlertID)

	assert.Nil(t, notificationWithoutAlert.AlertID)
}
