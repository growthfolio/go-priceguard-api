package services

import (
	"context"
	"fmt"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// WebSocketHub interface to avoid import cycle
type WebSocketHub interface {
	Broadcast(room, messageType string, data interface{})
	BroadcastToUser(userID uuid.UUID, messageType string, data interface{})
	GetConnectedClients() int
	GetRooms() map[string]int
}

// AlertWebSocketService integrates alert system with WebSocket broadcasting
type AlertWebSocketService struct {
	wsHub               WebSocketHub
	notificationService *NotificationService
	alertEngine         *AlertEngine
	logger              *logrus.Logger
}

// NewAlertWebSocketService creates a new alert WebSocket service
func NewAlertWebSocketService(
	wsHub WebSocketHub,
	notificationService *NotificationService,
	alertEngine *AlertEngine,
	logger *logrus.Logger,
) *AlertWebSocketService {
	return &AlertWebSocketService{
		wsHub:               wsHub,
		notificationService: notificationService,
		alertEngine:         alertEngine,
		logger:              logger,
	}
}

// BroadcastAlertTriggered broadcasts alert triggered events to connected clients
func (aws *AlertWebSocketService) BroadcastAlertTriggered(ctx context.Context, alert *entities.Alert, result *AlertEvaluationResult) error {
	// Create broadcast data
	data := map[string]interface{}{
		"alert_id":       alert.ID,
		"symbol":         alert.Symbol,
		"alert_type":     alert.AlertType,
		"condition_type": alert.ConditionType,
		"target_value":   alert.TargetValue,
		"current_value":  result.CurrentValue,
		"message":        result.Message,
		"timeframe":      alert.Timeframe,
		"triggered_at":   alert.TriggeredAt,
		"context":        result.Context,
	}

	// Broadcast to specific user
	aws.wsHub.BroadcastToUser(alert.UserID, "alert_triggered", data)

	aws.logger.WithFields(logrus.Fields{
		"alert_id": alert.ID,
		"user_id":  alert.UserID,
		"symbol":   alert.Symbol,
	}).Info("Alert triggered event broadcasted via WebSocket")

	return nil
}

// BroadcastNotificationUpdate broadcasts notification updates to connected clients
func (aws *AlertWebSocketService) BroadcastNotificationUpdate(ctx context.Context, notification *entities.Notification) error {
	// Create broadcast data
	data := map[string]interface{}{
		"notification_id":   notification.ID,
		"title":             notification.Title,
		"message":           notification.Message,
		"notification_type": notification.NotificationType,
		"alert_id":          notification.AlertID,
		"read_at":           notification.ReadAt,
		"created_at":        notification.CreatedAt,
	}

	// Broadcast to specific user
	aws.wsHub.BroadcastToUser(notification.UserID, "notification_update", data)

	aws.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"type":            notification.NotificationType,
	}).Info("Notification update broadcasted via WebSocket")

	return nil
}

// BroadcastCryptoDataUpdate broadcasts crypto data updates to subscribers
func (aws *AlertWebSocketService) BroadcastCryptoDataUpdate(ctx context.Context, symbol string, data map[string]interface{}) error {
	// Create symbol-specific room
	symbolRoom := fmt.Sprintf("crypto:%s", symbol)

	// Broadcast to symbol's room
	aws.wsHub.Broadcast(symbolRoom, "crypto_data_update", map[string]interface{}{
		"symbol": symbol,
		"data":   data,
	})

	aws.logger.WithFields(logrus.Fields{
		"symbol":      symbol,
		"symbol_room": symbolRoom,
	}).Debug("Crypto data update broadcasted via WebSocket")

	return nil
}

// BroadcastSystemAlert broadcasts system-wide alerts to all connected clients
func (aws *AlertWebSocketService) BroadcastSystemAlert(ctx context.Context, alertType, title, message string, data map[string]interface{}) error {
	// Create broadcast data
	broadcastData := map[string]interface{}{
		"alert_type": alertType,
		"title":      title,
		"message":    message,
		"data":       data,
	}

	// Broadcast to global system room
	aws.wsHub.Broadcast("system", "system_alert", broadcastData)

	aws.logger.WithFields(logrus.Fields{
		"alert_type": alertType,
		"title":      title,
	}).Info("System alert broadcasted via WebSocket")

	return nil
}

// NotifyAlertEvaluation sends real-time alert evaluation results
func (aws *AlertWebSocketService) NotifyAlertEvaluation(ctx context.Context, userID uuid.UUID, results []AlertEvaluationResult) error {
	if len(results) == 0 {
		return nil
	}

	// Count triggered alerts
	triggeredCount := 0
	for _, result := range results {
		if result.ShouldTrigger {
			triggeredCount++
		}
	}

	// Create broadcast data with evaluation summary
	data := map[string]interface{}{
		"user_id":          userID,
		"evaluation_count": len(results),
		"triggered_count":  triggeredCount,
		"results":          results,
	}

	// Broadcast to specific user
	aws.wsHub.BroadcastToUser(userID, "alert_evaluation", data)

	aws.logger.WithFields(logrus.Fields{
		"user_id":          userID,
		"evaluation_count": len(results),
		"triggered_count":  triggeredCount,
	}).Debug("Alert evaluation results broadcasted via WebSocket")

	return nil
}

// GetConnectedUsersStats returns statistics about connected users
func (aws *AlertWebSocketService) GetConnectedUsersStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"total_connections": aws.wsHub.GetConnectedClients(),
		"rooms":             aws.wsHub.GetRooms(),
	}

	return stats, nil
}
