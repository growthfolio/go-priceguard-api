package services

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
	"github.com/sirupsen/logrus"
)

// AlertCondition represents the different types of alert conditions
type AlertCondition string

const (
	ConditionPriceAbove     AlertCondition = "price_above"
	ConditionPriceBelow     AlertCondition = "price_below"
	ConditionRSIAbove       AlertCondition = "rsi_above"
	ConditionRSIBelow       AlertCondition = "rsi_below"
	ConditionPercentageUp   AlertCondition = "percentage_up"
	ConditionPercentageDown AlertCondition = "percentage_down"
	ConditionEMACrossUp     AlertCondition = "ema_cross_up"
	ConditionEMACrossDown   AlertCondition = "ema_cross_down"
	ConditionSMACrossUp     AlertCondition = "sma_cross_up"
	ConditionSMACrossDown   AlertCondition = "sma_cross_down"
)

// AlertEvaluationResult represents the result of evaluating an alert
type AlertEvaluationResult struct {
	AlertID       uuid.UUID              `json:"alert_id"`
	ShouldTrigger bool                   `json:"should_trigger"`
	CurrentValue  float64                `json:"current_value"`
	TargetValue   float64                `json:"target_value"`
	Message       string                 `json:"message"`
	Context       map[string]interface{} `json:"context"`
}

// AlertEngine handles the logic for evaluating and managing alerts
type AlertEngine struct {
	alertRepo                 repositories.AlertRepository
	priceHistoryRepo          repositories.PriceHistoryRepository
	technicalIndicatorRepo    repositories.TechnicalIndicatorRepository
	notificationRepo          repositories.NotificationRepository
	technicalIndicatorService *TechnicalIndicatorService
	webSocketService          AlertWebSocketService
	logger                    *logrus.Logger

	// Alert throttling
	throttleMap   map[uuid.UUID]time.Time
	throttleMutex sync.RWMutex

	// Alert state cache
	alertStateCache map[uuid.UUID]map[string]interface{}
	stateCacheMutex sync.RWMutex
}

// NewAlertEngine creates a new alert engine
func NewAlertEngine(
	alertRepo repositories.AlertRepository,
	priceHistoryRepo repositories.PriceHistoryRepository,
	technicalIndicatorRepo repositories.TechnicalIndicatorRepository,
	notificationRepo repositories.NotificationRepository,
	technicalIndicatorService *TechnicalIndicatorService,
	logger *logrus.Logger,
) *AlertEngine {
	return &AlertEngine{
		alertRepo:                 alertRepo,
		priceHistoryRepo:          priceHistoryRepo,
		technicalIndicatorRepo:    technicalIndicatorRepo,
		notificationRepo:          notificationRepo,
		technicalIndicatorService: technicalIndicatorService,
		logger:                    logger,
		throttleMap:               make(map[uuid.UUID]time.Time),
		alertStateCache:           make(map[uuid.UUID]map[string]interface{}),
	}
}

// SetWebSocketService sets the WebSocket service for broadcasting
func (ae *AlertEngine) SetWebSocketService(webSocketService AlertWebSocketService) {
	ae.webSocketService = webSocketService
}

// EvaluateAllAlerts evaluates all enabled alerts and triggers those that meet conditions
func (ae *AlertEngine) EvaluateAllAlerts(ctx context.Context) ([]AlertEvaluationResult, error) {
	alerts, err := ae.alertRepo.GetEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled alerts: %w", err)
	}

	var results []AlertEvaluationResult
	var wg sync.WaitGroup
	resultsChan := make(chan AlertEvaluationResult, len(alerts))

	// Evaluate alerts concurrently
	for _, alert := range alerts {
		wg.Add(1)
		go func(alert entities.Alert) {
			defer wg.Done()

			result, err := ae.EvaluateAlert(ctx, &alert)
			if err != nil {
				ae.logger.WithError(err).WithField("alert_id", alert.ID).Error("Failed to evaluate alert")
				return
			}

			if result != nil {
				resultsChan <- *result
			}
		}(alert)
	}

	// Wait for all evaluations to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results = append(results, result)
	}

	return results, nil
}

// EvaluateAlert evaluates a single alert and returns the result
func (ae *AlertEngine) EvaluateAlert(ctx context.Context, alert *entities.Alert) (*AlertEvaluationResult, error) {
	// Check if alert is throttled
	if ae.isThrottled(alert.ID) {
		return nil, nil
	}

	// Get current market data
	priceData, err := ae.priceHistoryRepo.GetLatest(ctx, alert.Symbol, alert.Timeframe)
	if err != nil {
		return nil, fmt.Errorf("failed to get price data for %s: %w", alert.Symbol, err)
	}

	if priceData == nil {
		return nil, fmt.Errorf("no price data available for %s", alert.Symbol)
	}

	// Evaluate based on alert type
	result, err := ae.evaluateCondition(ctx, alert, priceData)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate condition: %w", err)
	}

	// If alert should trigger, process it
	if result.ShouldTrigger {
		err := ae.processTriggeredAlert(ctx, alert, result)
		if err != nil {
			ae.logger.WithError(err).WithField("alert_id", alert.ID).Error("Failed to process triggered alert")
		}
	}

	return result, nil
}

// evaluateCondition evaluates the specific condition for an alert
func (ae *AlertEngine) evaluateCondition(ctx context.Context, alert *entities.Alert, priceData *entities.PriceHistory) (*AlertEvaluationResult, error) {
	result := &AlertEvaluationResult{
		AlertID:     alert.ID,
		TargetValue: alert.TargetValue,
		Context:     make(map[string]interface{}),
	}

	alertCondition := AlertCondition(alert.AlertType + "_" + alert.ConditionType)

	switch alertCondition {
	case ConditionPriceAbove:
		result.CurrentValue = priceData.ClosePrice
		result.ShouldTrigger = priceData.ClosePrice > alert.TargetValue
		result.Message = fmt.Sprintf("Price of %s is %.8f (target: %.8f)", alert.Symbol, priceData.ClosePrice, alert.TargetValue)

	case ConditionPriceBelow:
		result.CurrentValue = priceData.ClosePrice
		result.ShouldTrigger = priceData.ClosePrice < alert.TargetValue
		result.Message = fmt.Sprintf("Price of %s is %.8f (target: %.8f)", alert.Symbol, priceData.ClosePrice, alert.TargetValue)

	case ConditionPercentageUp, ConditionPercentageDown:
		return ae.evaluatePercentageChange(ctx, alert, priceData, result)

	case ConditionRSIAbove, ConditionRSIBelow:
		return ae.evaluateRSICondition(ctx, alert, priceData, result)

	case ConditionEMACrossUp, ConditionEMACrossDown, ConditionSMACrossUp, ConditionSMACrossDown:
		return ae.evaluateMovingAverageCross(ctx, alert, priceData, result)

	default:
		return nil, fmt.Errorf("unsupported alert condition: %s", alertCondition)
	}

	result.Context["price_data"] = map[string]interface{}{
		"open":   priceData.OpenPrice,
		"high":   priceData.HighPrice,
		"low":    priceData.LowPrice,
		"close":  priceData.ClosePrice,
		"volume": priceData.Volume,
	}

	return result, nil
}

// evaluatePercentageChange evaluates percentage change conditions
func (ae *AlertEngine) evaluatePercentageChange(ctx context.Context, alert *entities.Alert, currentPrice *entities.PriceHistory, result *AlertEvaluationResult) (*AlertEvaluationResult, error) {
	// Get price from 24 hours ago
	pastTime := currentPrice.Timestamp.Add(-24 * time.Hour)

	// Get historical data near that time
	historicalData, err := ae.priceHistoryRepo.GetBySymbol(ctx, alert.Symbol, alert.Timeframe, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical data: %w", err)
	}

	var basePrice float64
	minTimeDiff := time.Duration(math.MaxInt64)

	// Find the closest price to 24 hours ago
	for _, data := range historicalData {
		timeDiff := data.Timestamp.Sub(pastTime)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}
		if timeDiff < minTimeDiff {
			minTimeDiff = timeDiff
			basePrice = data.ClosePrice
		}
	}

	if basePrice == 0 {
		return nil, fmt.Errorf("no historical data found for percentage calculation")
	}

	percentageChange := ((currentPrice.ClosePrice - basePrice) / basePrice) * 100
	result.CurrentValue = percentageChange

	alertCondition := AlertCondition(alert.AlertType + "_" + alert.ConditionType)
	switch alertCondition {
	case ConditionPercentageUp:
		result.ShouldTrigger = percentageChange >= alert.TargetValue
		result.Message = fmt.Sprintf("%s gained %.2f%% in 24h (target: %.2f%%)", alert.Symbol, percentageChange, alert.TargetValue)
	case ConditionPercentageDown:
		result.ShouldTrigger = percentageChange <= -alert.TargetValue
		result.Message = fmt.Sprintf("%s lost %.2f%% in 24h (target: %.2f%%)", alert.Symbol, math.Abs(percentageChange), alert.TargetValue)
	}

	result.Context["base_price"] = basePrice
	result.Context["current_price"] = currentPrice.ClosePrice
	result.Context["percentage_change"] = percentageChange

	return result, nil
}

// evaluateRSICondition evaluates RSI-based conditions
func (ae *AlertEngine) evaluateRSICondition(ctx context.Context, alert *entities.Alert, priceData *entities.PriceHistory, result *AlertEvaluationResult) (*AlertEvaluationResult, error) {
	// Get latest RSI indicator
	rsiIndicator, err := ae.technicalIndicatorRepo.GetLatest(ctx, alert.Symbol, alert.Timeframe, "rsi")
	if err != nil {
		return nil, fmt.Errorf("failed to get RSI indicator: %w", err)
	}

	if rsiIndicator == nil || rsiIndicator.Value == nil {
		return nil, fmt.Errorf("no RSI data available for %s", alert.Symbol)
	}

	rsiValue := *rsiIndicator.Value
	result.CurrentValue = rsiValue

	alertCondition := AlertCondition(alert.AlertType + "_" + alert.ConditionType)
	switch alertCondition {
	case ConditionRSIAbove:
		result.ShouldTrigger = rsiValue > alert.TargetValue
		result.Message = fmt.Sprintf("RSI of %s is %.2f (target: above %.2f)", alert.Symbol, rsiValue, alert.TargetValue)
	case ConditionRSIBelow:
		result.ShouldTrigger = rsiValue < alert.TargetValue
		result.Message = fmt.Sprintf("RSI of %s is %.2f (target: below %.2f)", alert.Symbol, rsiValue, alert.TargetValue)
	}

	result.Context["rsi_value"] = rsiValue
	result.Context["indicator_timestamp"] = rsiIndicator.Timestamp

	return result, nil
}

// evaluateMovingAverageCross evaluates moving average crossover conditions
func (ae *AlertEngine) evaluateMovingAverageCross(ctx context.Context, alert *entities.Alert, priceData *entities.PriceHistory, result *AlertEvaluationResult) (*AlertEvaluationResult, error) {
	// For MA crosses, we need to check if there was a crossover
	// This requires comparing current and previous states

	// Get state from cache
	ae.stateCacheMutex.RLock()
	previousState, exists := ae.alertStateCache[alert.ID]
	ae.stateCacheMutex.RUnlock()

	// Get short and long period MAs (assuming target value represents the short period)
	shortPeriod := int(alert.TargetValue)
	longPeriod := shortPeriod * 2 // Long period is double the short period

	var indicatorType string
	alertCondition := AlertCondition(alert.AlertType + "_" + alert.ConditionType)

	switch alertCondition {
	case ConditionEMACrossUp, ConditionEMACrossDown:
		indicatorType = "ema"
	case ConditionSMACrossUp, ConditionSMACrossDown:
		indicatorType = "sma"
	default:
		return nil, fmt.Errorf("unsupported MA cross condition: %s", alertCondition)
	}

	// Get current MAs
	shortMA, err := ae.technicalIndicatorRepo.GetLatest(ctx, alert.Symbol, alert.Timeframe, fmt.Sprintf("%s_%d", indicatorType, shortPeriod))
	if err != nil {
		return nil, fmt.Errorf("failed to get short %s: %w", indicatorType, err)
	}

	longMA, err := ae.technicalIndicatorRepo.GetLatest(ctx, alert.Symbol, alert.Timeframe, fmt.Sprintf("%s_%d", indicatorType, longPeriod))
	if err != nil {
		return nil, fmt.Errorf("failed to get long %s: %w", indicatorType, err)
	}

	if shortMA == nil || longMA == nil || shortMA.Value == nil || longMA.Value == nil {
		return nil, fmt.Errorf("insufficient MA data for %s", alert.Symbol)
	}

	currentShort := *shortMA.Value
	currentLong := *longMA.Value

	// Update current state
	currentState := map[string]interface{}{
		"short_ma":  currentShort,
		"long_ma":   currentLong,
		"timestamp": time.Now(),
	}

	ae.stateCacheMutex.Lock()
	ae.alertStateCache[alert.ID] = currentState
	ae.stateCacheMutex.Unlock()

	result.CurrentValue = currentShort - currentLong

	if !exists {
		// First evaluation, no crossover yet
		result.ShouldTrigger = false
		result.Message = fmt.Sprintf("Monitoring %s crossover for %s", indicatorType, alert.Symbol)
	} else {
		// Check for crossover
		prevShort, _ := previousState["short_ma"].(float64)
		prevLong, _ := previousState["long_ma"].(float64)

		wasBelowPreviously := prevShort < prevLong
		isAboveNow := currentShort > currentLong

		switch alertCondition {
		case ConditionEMACrossUp, ConditionSMACrossUp:
			result.ShouldTrigger = wasBelowPreviously && isAboveNow
			if result.ShouldTrigger {
				result.Message = fmt.Sprintf("%s crossed above %s for %s",
					fmt.Sprintf("%s(%d)", indicatorType, shortPeriod),
					fmt.Sprintf("%s(%d)", indicatorType, longPeriod),
					alert.Symbol)
			}
		case ConditionEMACrossDown, ConditionSMACrossDown:
			result.ShouldTrigger = !wasBelowPreviously && !isAboveNow
			if result.ShouldTrigger {
				result.Message = fmt.Sprintf("%s crossed below %s for %s",
					fmt.Sprintf("%s(%d)", indicatorType, shortPeriod),
					fmt.Sprintf("%s(%d)", indicatorType, longPeriod),
					alert.Symbol)
			}
		}
	}

	result.Context["short_ma"] = currentShort
	result.Context["long_ma"] = currentLong
	result.Context["short_period"] = shortPeriod
	result.Context["long_period"] = longPeriod

	return result, nil
}

// processTriggeredAlert handles the actions when an alert is triggered
func (ae *AlertEngine) processTriggeredAlert(ctx context.Context, alert *entities.Alert, result *AlertEvaluationResult) error {
	// Update alert with triggered timestamp
	now := time.Now()
	alert.TriggeredAt = &now

	if err := ae.alertRepo.Update(ctx, alert); err != nil {
		return fmt.Errorf("failed to update alert: %w", err)
	}

	// Create notification
	notification := &entities.Notification{
		ID:               uuid.New(),
		UserID:           alert.UserID,
		AlertID:          &alert.ID,
		Title:            "Alert Triggered",
		Message:          result.Message,
		NotificationType: "alert_triggered",
		CreatedAt:        now,
	}

	if err := ae.notificationRepo.Create(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Broadcast via WebSocket if service is available
	if ae.webSocketService != nil {
		// Broadcast alert triggered event
		if err := ae.webSocketService.BroadcastAlertTriggered(ctx, alert, result); err != nil {
			ae.logger.WithError(err).Warn("Failed to broadcast alert triggered event")
		}

		// Broadcast notification update
		if err := ae.webSocketService.BroadcastNotificationUpdate(ctx, notification); err != nil {
			ae.logger.WithError(err).Warn("Failed to broadcast notification update")
		}
	}

	// Set throttle to prevent spam
	ae.setThrottle(alert.ID, 5*time.Minute) // 5 minute throttle

	ae.logger.WithFields(logrus.Fields{
		"alert_id":      alert.ID,
		"user_id":       alert.UserID,
		"symbol":        alert.Symbol,
		"alert_type":    alert.AlertType,
		"condition":     alert.ConditionType,
		"current_value": result.CurrentValue,
		"target_value":  result.TargetValue,
	}).Info("Alert triggered successfully")

	return nil
}

// isThrottled checks if an alert is currently throttled
func (ae *AlertEngine) isThrottled(alertID uuid.UUID) bool {
	ae.throttleMutex.RLock()
	defer ae.throttleMutex.RUnlock()

	throttleTime, exists := ae.throttleMap[alertID]
	if !exists {
		return false
	}

	return time.Now().Before(throttleTime)
}

// setThrottle sets a throttle period for an alert
func (ae *AlertEngine) setThrottle(alertID uuid.UUID, duration time.Duration) {
	ae.throttleMutex.Lock()
	defer ae.throttleMutex.Unlock()

	ae.throttleMap[alertID] = time.Now().Add(duration)
}

// CleanupThrottles removes expired throttles
func (ae *AlertEngine) CleanupThrottles() {
	ae.throttleMutex.Lock()
	defer ae.throttleMutex.Unlock()

	now := time.Now()
	for alertID, throttleTime := range ae.throttleMap {
		if now.After(throttleTime) {
			delete(ae.throttleMap, alertID)
		}
	}
}

// GetAlertStats returns statistics about alert evaluations
func (ae *AlertEngine) GetAlertStats(ctx context.Context) (map[string]interface{}, error) {
	alerts, err := ae.alertRepo.GetEnabled(ctx)
	if err != nil {
		return nil, err
	}

	ae.throttleMutex.RLock()
	throttledCount := len(ae.throttleMap)
	ae.throttleMutex.RUnlock()

	ae.stateCacheMutex.RLock()
	cachedStatesCount := len(ae.alertStateCache)
	ae.stateCacheMutex.RUnlock()

	stats := map[string]interface{}{
		"total_enabled_alerts": len(alerts),
		"throttled_alerts":     throttledCount,
		"cached_alert_states":  cachedStatesCount,
		"last_update":          time.Now(),
	}

	return stats, nil
}
