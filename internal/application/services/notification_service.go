package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/repositories"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// NotificationChannel represents different notification delivery channels
type NotificationChannel string

const (
	ChannelInApp NotificationChannel = "app"
	ChannelEmail NotificationChannel = "email"
	ChannelPush  NotificationChannel = "push"
	ChannelSMS   NotificationChannel = "sms"
)

// NotificationPriority represents the priority of notifications
type NotificationPriority string

const (
	PriorityLow    NotificationPriority = "low"
	PriorityNormal NotificationPriority = "normal"
	PriorityHigh   NotificationPriority = "high"
	PriorityUrgent NotificationPriority = "urgent"
)

// QueuedNotification represents a notification queued for processing
type QueuedNotification struct {
	ID          uuid.UUID              `json:"id"`
	UserID      uuid.UUID              `json:"user_id"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Channels    []NotificationChannel  `json:"channels"`
	Priority    NotificationPriority   `json:"priority"`
	Data        map[string]interface{} `json:"data,omitempty"`
	ScheduledAt time.Time              `json:"scheduled_at"`
	CreatedAt   time.Time              `json:"created_at"`
	Retries     int                    `json:"retries"`
	MaxRetries  int                    `json:"max_retries"`
}

// NotificationDeliveryResult represents the result of a notification delivery attempt
type NotificationDeliveryResult struct {
	NotificationID uuid.UUID           `json:"notification_id"`
	Channel        NotificationChannel `json:"channel"`
	Success        bool                `json:"success"`
	Error          string              `json:"error,omitempty"`
	DeliveredAt    time.Time           `json:"delivered_at"`
}

// NotificationService handles notification creation, queuing, and delivery
type NotificationService struct {
	notificationRepo repositories.NotificationRepository
	userRepo         repositories.UserRepository
	redisClient      *redis.Client
	logger           *logrus.Logger

	// Processing control
	isProcessing bool
	stopChan     chan struct{}
	processingWG sync.WaitGroup
	mutex        sync.RWMutex

	// Configuration
	queueKey          string
	dlqKey            string // Dead Letter Queue
	processingTimeout time.Duration
	batchSize         int
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	notificationRepo repositories.NotificationRepository,
	userRepo repositories.UserRepository,
	redisClient *redis.Client,
	logger *logrus.Logger,
) *NotificationService {
	return &NotificationService{
		notificationRepo:  notificationRepo,
		userRepo:          userRepo,
		redisClient:       redisClient,
		logger:            logger,
		queueKey:          "notification_queue",
		dlqKey:            "notification_dlq",
		processingTimeout: 30 * time.Second,
		batchSize:         10,
		stopChan:          make(chan struct{}),
	}
}

// StartProcessing starts the notification processing worker
func (ns *NotificationService) StartProcessing(ctx context.Context) {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if ns.isProcessing {
		ns.logger.Warn("Notification service is already processing")
		return
	}

	ns.isProcessing = true
	ns.logger.Info("Starting notification processing")

	ns.processingWG.Add(1)
	go ns.processNotificationQueue(ctx)
}

// StopProcessing stops the notification processing worker
func (ns *NotificationService) StopProcessing() {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if !ns.isProcessing {
		return
	}

	ns.logger.Info("Stopping notification processing")
	close(ns.stopChan)
	ns.processingWG.Wait()
	ns.isProcessing = false
}

// CreateNotification creates a new in-app notification
func (ns *NotificationService) CreateNotification(ctx context.Context, userID uuid.UUID, notificationType, title, message string, data map[string]interface{}) (*entities.Notification, error) {
	notification := &entities.Notification{
		ID:               uuid.New(),
		UserID:           userID,
		Title:            title,
		Message:          message,
		NotificationType: notificationType,
		CreatedAt:        time.Now(),
	}

	if err := ns.notificationRepo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	ns.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         userID,
		"type":            notificationType,
	}).Info("Notification created")

	return notification, nil
}

// QueueNotification queues a notification for processing across multiple channels
func (ns *NotificationService) QueueNotification(ctx context.Context, notification *QueuedNotification) error {
	// Set defaults
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	if notification.ScheduledAt.IsZero() {
		notification.ScheduledAt = time.Now()
	}
	if notification.MaxRetries == 0 {
		notification.MaxRetries = 3
	}
	if notification.Priority == "" {
		notification.Priority = PriorityNormal
	}

	// Serialize notification
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to serialize notification: %w", err)
	}

	// Add to Redis queue with priority
	score := float64(notification.ScheduledAt.Unix())
	if notification.Priority == PriorityUrgent {
		score -= 86400 // Move urgent notifications 24h earlier in queue
	} else if notification.Priority == PriorityHigh {
		score -= 3600 // Move high priority 1h earlier
	}

	err = ns.redisClient.ZAdd(ctx, ns.queueKey, redis.Z{
		Score:  score,
		Member: string(data),
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to queue notification: %w", err)
	}

	ns.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"priority":        notification.Priority,
		"channels":        notification.Channels,
		"scheduled_at":    notification.ScheduledAt,
	}).Info("Notification queued")

	return nil
}

// QueueAlertNotification is a convenience method for queuing alert-related notifications
func (ns *NotificationService) QueueAlertNotification(ctx context.Context, alert *entities.Alert, currentValue float64, channels []NotificationChannel) error {
	// Get user to check preferences (for future use)
	_, err := ns.userRepo.GetByID(ctx, alert.UserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Create notification message
	title := "Price Alert Triggered"
	message := fmt.Sprintf("Your alert for %s has been triggered. Current value: %.8f (Target: %.8f)",
		alert.Symbol, currentValue, alert.TargetValue)

	// Prepare notification data
	data := map[string]interface{}{
		"alert_id":      alert.ID,
		"symbol":        alert.Symbol,
		"alert_type":    alert.AlertType,
		"condition":     alert.ConditionType,
		"target_value":  alert.TargetValue,
		"current_value": currentValue,
		"timeframe":     alert.Timeframe,
	}

	// Create queued notification
	queuedNotification := &QueuedNotification{
		UserID:   alert.UserID,
		Type:     "alert_triggered",
		Title:    title,
		Message:  message,
		Channels: channels,
		Priority: PriorityHigh,
		Data:     data,
	}

	// First create in-app notification
	_, err = ns.CreateNotification(ctx, alert.UserID, "alert_triggered", title, message, data)
	if err != nil {
		ns.logger.WithError(err).Error("Failed to create in-app notification")
	}

	// Queue for other channels if enabled
	if len(channels) > 1 || (len(channels) == 1 && channels[0] != ChannelInApp) {
		return ns.QueueNotification(ctx, queuedNotification)
	}

	return nil
}

// processNotificationQueue processes notifications from the queue
func (ns *NotificationService) processNotificationQueue(ctx context.Context) {
	defer ns.processingWG.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ns.stopChan:
			return
		case <-ticker.C:
			ns.processBatch(ctx)
		}
	}
}

// processBatch processes a batch of notifications from the queue
func (ns *NotificationService) processBatch(ctx context.Context) {
	// Get notifications ready for processing
	now := time.Now().Unix()
	results, err := ns.redisClient.ZRangeByScoreWithScores(ctx, ns.queueKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%d", now),
		Count: int64(ns.batchSize),
	}).Result()

	if err != nil {
		ns.logger.WithError(err).Error("Failed to get notifications from queue")
		return
	}

	if len(results) == 0 {
		return
	}

	ns.logger.WithField("count", len(results)).Debug("Processing notification batch")

	for _, result := range results {
		notificationData := result.Member.(string)

		// Parse notification
		var notification QueuedNotification
		if err := json.Unmarshal([]byte(notificationData), &notification); err != nil {
			ns.logger.WithError(err).Error("Failed to parse queued notification")
			ns.moveToDeadLetterQueue(ctx, notificationData, "parse_error")
			ns.removeFromQueue(ctx, notificationData)
			continue
		}

		// Process notification
		success := ns.processNotification(ctx, &notification)

		if success {
			// Remove from queue on success
			ns.removeFromQueue(ctx, notificationData)
		} else {
			// Handle retry logic
			notification.Retries++
			if notification.Retries >= notification.MaxRetries {
				// Move to dead letter queue
				ns.moveToDeadLetterQueue(ctx, notificationData, "max_retries_exceeded")
				ns.removeFromQueue(ctx, notificationData)
			} else {
				// Reschedule with exponential backoff
				backoffDelay := time.Duration(notification.Retries*notification.Retries) * time.Minute
				notification.ScheduledAt = time.Now().Add(backoffDelay)

				// Remove old entry and add new one
				ns.removeFromQueue(ctx, notificationData)
				ns.QueueNotification(ctx, &notification)
			}
		}
	}
}

// processNotification processes a single notification across all its channels
func (ns *NotificationService) processNotification(ctx context.Context, notification *QueuedNotification) bool {
	allSuccess := true

	for _, channel := range notification.Channels {
		result := ns.deliverToChannel(ctx, notification, channel)

		if !result.Success {
			allSuccess = false
			ns.logger.WithFields(logrus.Fields{
				"notification_id": notification.ID,
				"channel":         channel,
				"error":           result.Error,
			}).Error("Failed to deliver notification")
		}
	}

	return allSuccess
}

// deliverToChannel delivers a notification to a specific channel
func (ns *NotificationService) deliverToChannel(ctx context.Context, notification *QueuedNotification, channel NotificationChannel) *NotificationDeliveryResult {
	result := &NotificationDeliveryResult{
		NotificationID: notification.ID,
		Channel:        channel,
		DeliveredAt:    time.Now(),
	}

	switch channel {
	case ChannelInApp:
		// In-app notifications are already created, just mark as delivered
		result.Success = true

	case ChannelEmail:
		// Email delivery would be implemented here
		// For now, just simulate
		result.Success = ns.simulateEmailDelivery(notification)
		if !result.Success {
			result.Error = "email delivery failed"
		}

	case ChannelPush:
		// Push notification delivery would be implemented here
		result.Success = ns.simulatePushDelivery(notification)
		if !result.Success {
			result.Error = "push delivery failed"
		}

	case ChannelSMS:
		// SMS delivery would be implemented here
		result.Success = ns.simulateSMSDelivery(notification)
		if !result.Success {
			result.Error = "sms delivery failed"
		}

	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported channel: %s", channel)
	}

	return result
}

// simulateEmailDelivery simulates email delivery (placeholder for actual implementation)
func (ns *NotificationService) simulateEmailDelivery(notification *QueuedNotification) bool {
	ns.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"title":           notification.Title,
	}).Info("Email notification would be sent here")
	return true // Simulate success
}

// simulatePushDelivery simulates push notification delivery
func (ns *NotificationService) simulatePushDelivery(notification *QueuedNotification) bool {
	ns.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"title":           notification.Title,
	}).Info("Push notification would be sent here")
	return true // Simulate success
}

// simulateSMSDelivery simulates SMS delivery
func (ns *NotificationService) simulateSMSDelivery(notification *QueuedNotification) bool {
	ns.logger.WithFields(logrus.Fields{
		"notification_id": notification.ID,
		"user_id":         notification.UserID,
		"message":         notification.Message,
	}).Info("SMS notification would be sent here")
	return true // Simulate success
}

// removeFromQueue removes a notification from the processing queue
func (ns *NotificationService) removeFromQueue(ctx context.Context, notificationData string) {
	err := ns.redisClient.ZRem(ctx, ns.queueKey, notificationData).Err()
	if err != nil {
		ns.logger.WithError(err).Error("Failed to remove notification from queue")
	}
}

// moveToDeadLetterQueue moves a notification to the dead letter queue
func (ns *NotificationService) moveToDeadLetterQueue(ctx context.Context, notificationData, reason string) {
	dlqData := map[string]interface{}{
		"notification": notificationData,
		"reason":       reason,
		"timestamp":    time.Now(),
	}

	data, err := json.Marshal(dlqData)
	if err != nil {
		ns.logger.WithError(err).Error("Failed to serialize DLQ entry")
		return
	}

	err = ns.redisClient.ZAdd(ctx, ns.dlqKey, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: string(data),
	}).Err()

	if err != nil {
		ns.logger.WithError(err).Error("Failed to add to dead letter queue")
	}
}

// GetNotificationStats returns statistics about the notification system
func (ns *NotificationService) GetNotificationStats(ctx context.Context) (map[string]interface{}, error) {
	queueSize, err := ns.redisClient.ZCard(ctx, ns.queueKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get queue size: %w", err)
	}

	dlqSize, err := ns.redisClient.ZCard(ctx, ns.dlqKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ size: %w", err)
	}

	stats := map[string]interface{}{
		"queue_size":    queueSize,
		"dlq_size":      dlqSize,
		"is_processing": ns.isProcessing,
		"last_update":   time.Now(),
	}

	return stats, nil
}

// CleanupOldNotifications removes old processed notifications and DLQ entries
func (ns *NotificationService) CleanupOldNotifications(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan).Unix()

	// Clean up DLQ
	removed, err := ns.redisClient.ZRemRangeByScore(ctx, ns.dlqKey, "-inf", fmt.Sprintf("%d", cutoff)).Result()
	if err != nil {
		return fmt.Errorf("failed to cleanup DLQ: %w", err)
	}

	ns.logger.WithField("removed_count", removed).Info("Cleaned up old DLQ entries")
	return nil
}
