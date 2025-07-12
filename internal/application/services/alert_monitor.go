package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/repositories"
	"github.com/sirupsen/logrus"
)

// AlertMonitor manages the real-time monitoring of alerts
type AlertMonitor struct {
	alertEngine         *AlertEngine
	notificationService *NotificationService
	cryptoDataService   *CryptoDataService
	alertRepo           repositories.AlertRepository
	logger              *logrus.Logger

	// Monitoring control
	isRunning    bool
	stopChan     chan struct{}
	monitoringWG sync.WaitGroup
	mutex        sync.RWMutex

	// Configuration
	evaluationInterval time.Duration
	cleanupInterval    time.Duration
}

// NewAlertMonitor creates a new alert monitor
func NewAlertMonitor(
	alertEngine *AlertEngine,
	notificationService *NotificationService,
	cryptoDataService *CryptoDataService,
	alertRepo repositories.AlertRepository,
	logger *logrus.Logger,
) *AlertMonitor {
	return &AlertMonitor{
		alertEngine:         alertEngine,
		notificationService: notificationService,
		cryptoDataService:   cryptoDataService,
		alertRepo:           alertRepo,
		logger:              logger,
		stopChan:            make(chan struct{}),
		evaluationInterval:  30 * time.Second, // Evaluate alerts every 30 seconds
		cleanupInterval:     5 * time.Minute,  // Cleanup throttles every 5 minutes
	}
}

// Start begins the alert monitoring process
func (am *AlertMonitor) Start(ctx context.Context) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if am.isRunning {
		am.logger.Warn("Alert monitor is already running")
		return
	}

	am.isRunning = true
	am.logger.Info("Starting alert monitor")

	// Start evaluation worker
	am.monitoringWG.Add(1)
	go am.evaluationWorker(ctx)

	// Start cleanup worker
	am.monitoringWG.Add(1)
	go am.cleanupWorker(ctx)
}

// Stop stops the alert monitoring process
func (am *AlertMonitor) Stop() {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if !am.isRunning {
		return
	}

	am.logger.Info("Stopping alert monitor")
	close(am.stopChan)
	am.monitoringWG.Wait()
	am.isRunning = false
}

// IsRunning returns whether the monitor is currently running
func (am *AlertMonitor) IsRunning() bool {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.isRunning
}

// evaluationWorker continuously evaluates alerts
func (am *AlertMonitor) evaluationWorker(ctx context.Context) {
	defer am.monitoringWG.Done()

	ticker := time.NewTicker(am.evaluationInterval)
	defer ticker.Stop()

	am.logger.Info("Alert evaluation worker started")

	for {
		select {
		case <-ctx.Done():
			am.logger.Info("Alert evaluation worker stopped due to context cancellation")
			return
		case <-am.stopChan:
			am.logger.Info("Alert evaluation worker stopped")
			return
		case <-ticker.C:
			am.performEvaluation(ctx)
		}
	}
}

// cleanupWorker periodically cleans up throttles and old data
func (am *AlertMonitor) cleanupWorker(ctx context.Context) {
	defer am.monitoringWG.Done()

	ticker := time.NewTicker(am.cleanupInterval)
	defer ticker.Stop()

	am.logger.Info("Alert cleanup worker started")

	for {
		select {
		case <-ctx.Done():
			am.logger.Info("Alert cleanup worker stopped due to context cancellation")
			return
		case <-am.stopChan:
			am.logger.Info("Alert cleanup worker stopped")
			return
		case <-ticker.C:
			am.performCleanup(ctx)
		}
	}
}

// performEvaluation evaluates all enabled alerts
func (am *AlertMonitor) performEvaluation(ctx context.Context) {
	start := time.Now()

	results, err := am.alertEngine.EvaluateAllAlerts(ctx)
	if err != nil {
		am.logger.WithError(err).Error("Failed to evaluate alerts")
		return
	}

	triggeredCount := 0
	for _, result := range results {
		if result.ShouldTrigger {
			triggeredCount++

			// Get the alert for notification processing
			alert, err := am.alertRepo.GetByID(ctx, result.AlertID)
			if err != nil {
				am.logger.WithError(err).WithField("alert_id", result.AlertID).Error("Failed to get alert for notification")
				continue
			}

			// Queue notification based on alert preferences
			channels := make([]NotificationChannel, 0, len(alert.NotifyVia))
			for _, channel := range alert.NotifyVia {
				switch channel {
				case "app":
					channels = append(channels, ChannelInApp)
				case "email":
					channels = append(channels, ChannelEmail)
				case "push":
					channels = append(channels, ChannelPush)
				case "sms":
					channels = append(channels, ChannelSMS)
				}
			}

			// Queue the notification
			err = am.notificationService.QueueAlertNotification(ctx, alert, result.CurrentValue, channels)
			if err != nil {
				am.logger.WithError(err).WithField("alert_id", result.AlertID).Error("Failed to queue alert notification")
			}
		}
	}

	duration := time.Since(start)
	am.logger.WithFields(logrus.Fields{
		"evaluation_time": duration,
		"total_alerts":    len(results),
		"triggered_count": triggeredCount,
	}).Debug("Alert evaluation completed")
}

// performCleanup cleans up throttles and old data
func (am *AlertMonitor) performCleanup(ctx context.Context) {
	// Cleanup alert throttles
	am.alertEngine.CleanupThrottles()

	// Cleanup old notifications (older than 30 days)
	err := am.notificationService.CleanupOldNotifications(ctx, 30*24*time.Hour)
	if err != nil {
		am.logger.WithError(err).Error("Failed to cleanup old notifications")
	}

	am.logger.Debug("Alert monitor cleanup completed")
}

// TriggerImmediateEvaluation triggers an immediate evaluation of all alerts
func (am *AlertMonitor) TriggerImmediateEvaluation(ctx context.Context) error {
	if !am.IsRunning() {
		return fmt.Errorf("alert monitor is not running")
	}

	go am.performEvaluation(ctx)
	return nil
}

// GetMonitorStats returns statistics about the alert monitor
func (am *AlertMonitor) GetMonitorStats(ctx context.Context) (map[string]interface{}, error) {
	alertStats, err := am.alertEngine.GetAlertStats(ctx)
	if err != nil {
		return nil, err
	}

	notificationStats, err := am.notificationService.GetNotificationStats(ctx)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"monitor_running":     am.IsRunning(),
		"evaluation_interval": am.evaluationInterval.String(),
		"cleanup_interval":    am.cleanupInterval.String(),
		"alert_engine_stats":  alertStats,
		"notification_stats":  notificationStats,
		"last_update":         time.Now(),
	}

	return stats, nil
}
