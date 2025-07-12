package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/felipe-macedo/go-priceguard-api/internal/application/services"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/repositories"
)

type AlertHandler struct {
	alertRepo    repositories.AlertRepository
	alertMonitor *services.AlertMonitor
	alertEngine  *services.AlertEngine
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(
	alertRepo repositories.AlertRepository,
	alertMonitor *services.AlertMonitor,
	alertEngine *services.AlertEngine,
) *AlertHandler {
	return &AlertHandler{
		alertRepo:    alertRepo,
		alertMonitor: alertMonitor,
		alertEngine:  alertEngine,
	}
}

// GetAlerts godoc
// @Summary Get user alerts
// @Description Get list of alerts for the authenticated user
// @Tags Alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/alerts [get]
func (h *AlertHandler) GetAlerts(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	// Validate limits
	if limit > 100 {
		limit = 100
	}
	if limit <= 0 {
		limit = 50
	}

	alerts, err := h.alertRepo.GetByUserID(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   alerts,
		"limit":  limit,
		"offset": offset,
		"count":  len(alerts),
	})
}

// CreateAlert godoc
// @Summary Create new alert
// @Description Create a new price alert for the authenticated user
// @Tags Alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param alert body entities.Alert true "Alert data"
// @Success 201 {object} entities.Alert
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/alerts [post]
func (h *AlertHandler) CreateAlert(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var alertData struct {
		Symbol        string   `json:"symbol" binding:"required"`
		AlertType     string   `json:"alert_type" binding:"required"`
		ConditionType string   `json:"condition_type" binding:"required"`
		TargetValue   float64  `json:"target_value" binding:"required"`
		Timeframe     string   `json:"timeframe" binding:"required"`
		NotifyVia     []string `json:"notify_via,omitempty"`
		Enabled       *bool    `json:"enabled,omitempty"`
	}

	if err := c.ShouldBindJSON(&alertData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Validate condition type
	validConditions := map[string]bool{
		"above": true, "below": true, "crosses_up": true, "crosses_down": true,
	}
	if !validConditions[alertData.ConditionType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid condition type"})
		return
	}

	// Set default notify_via if not provided
	notifyVia := alertData.NotifyVia
	if len(notifyVia) == 0 {
		notifyVia = []string{"app"}
	}

	// Create alert
	alert := &entities.Alert{
		UserID:        userID.(uuid.UUID),
		Symbol:        alertData.Symbol,
		AlertType:     alertData.AlertType,
		ConditionType: alertData.ConditionType,
		TargetValue:   alertData.TargetValue,
		Timeframe:     alertData.Timeframe,
		Enabled:       alertData.Enabled == nil || *alertData.Enabled,
		NotifyVia:     notifyVia,
	}

	if err := h.alertRepo.Create(c.Request.Context(), alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert"})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

// UpdateAlert godoc
// @Summary Update alert
// @Description Update an existing alert for the authenticated user
// @Tags Alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Alert ID"
// @Param alert body entities.Alert true "Alert data"
// @Success 200 {object} entities.Alert
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Alert not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/alerts/{id} [put]
func (h *AlertHandler) UpdateAlert(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	alertIDStr := c.Param("id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	// Get existing alert
	alert, err := h.alertRepo.GetByID(c.Request.Context(), alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	// Check if user owns the alert
	if alert.UserID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var updateData struct {
		AlertType     *string   `json:"alert_type,omitempty"`
		ConditionType *string   `json:"condition_type,omitempty"`
		TargetValue   *float64  `json:"target_value,omitempty"`
		Timeframe     *string   `json:"timeframe,omitempty"`
		NotifyVia     *[]string `json:"notify_via,omitempty"`
		Enabled       *bool     `json:"enabled,omitempty"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Update fields if provided
	if updateData.AlertType != nil {
		alert.AlertType = *updateData.AlertType
	}
	if updateData.ConditionType != nil {
		alert.ConditionType = *updateData.ConditionType
	}
	if updateData.TargetValue != nil {
		alert.TargetValue = *updateData.TargetValue
	}
	if updateData.Timeframe != nil {
		alert.Timeframe = *updateData.Timeframe
	}
	if updateData.NotifyVia != nil {
		alert.NotifyVia = *updateData.NotifyVia
	}
	if updateData.Enabled != nil {
		alert.Enabled = *updateData.Enabled
	}

	if err := h.alertRepo.Update(c.Request.Context(), alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update alert"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// DeleteAlert godoc
// @Summary Delete alert
// @Description Delete an alert for the authenticated user
// @Tags Alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Alert ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Alert not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/alerts/{id} [delete]
func (h *AlertHandler) DeleteAlert(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	alertIDStr := c.Param("id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	// Get existing alert to check ownership
	alert, err := h.alertRepo.GetByID(c.Request.Context(), alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	// Check if user owns the alert
	if alert.UserID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.alertRepo.Delete(c.Request.Context(), alertID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alert"})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetAlertStats godoc
// @Summary Get alert system statistics
// @Description Get statistics about the alert monitoring system
// @Tags Alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/alerts/stats [get]
func (h *AlertHandler) GetAlertStats(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	stats, err := h.alertMonitor.GetMonitorStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get alert stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// TriggerEvaluation godoc
// @Summary Trigger immediate alert evaluation
// @Description Trigger an immediate evaluation of all alerts (admin only)
// @Tags Alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/alerts/trigger-evaluation [post]
func (h *AlertHandler) TriggerEvaluation(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err := h.alertMonitor.TriggerImmediateEvaluation(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger evaluation", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Alert evaluation triggered",
		"triggered_at": time.Now(),
	})
}

// EvaluateAlert godoc
// @Summary Evaluate a specific alert
// @Description Evaluate a specific alert and return the result
// @Tags Alerts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Alert ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Alert not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/alerts/{id}/evaluate [post]
func (h *AlertHandler) EvaluateAlert(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	alertIDStr := c.Param("id")
	alertID, err := uuid.Parse(alertIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid alert ID"})
		return
	}

	// Get the alert
	alert, err := h.alertRepo.GetByID(c.Request.Context(), alertID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Alert not found"})
		return
	}

	// Check if user owns the alert
	if alert.UserID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Evaluate the alert
	result, err := h.alertEngine.EvaluateAlert(c.Request.Context(), alert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to evaluate alert", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"alert_id":     alertID,
		"evaluation":   result,
		"evaluated_at": time.Now(),
	})
}

// GetAlertTypes godoc
// @Summary Get available alert types and conditions
// @Description Get list of available alert types and their supported conditions
// @Tags Alerts
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/alerts/types [get]
func (h *AlertHandler) GetAlertTypes(c *gin.Context) {
	alertTypes := map[string]interface{}{
		"price": map[string]interface{}{
			"description":    "Price-based alerts",
			"conditions":     []string{"above", "below"},
			"example_target": 50000.0,
		},
		"percentage": map[string]interface{}{
			"description":    "Percentage change alerts",
			"conditions":     []string{"up", "down"},
			"example_target": 5.0,
		},
		"rsi": map[string]interface{}{
			"description":    "RSI indicator alerts",
			"conditions":     []string{"above", "below"},
			"example_target": 70.0,
		},
		"ema_cross": map[string]interface{}{
			"description":    "EMA crossover alerts",
			"conditions":     []string{"up", "down"},
			"example_target": 20.0,
		},
		"sma_cross": map[string]interface{}{
			"description":    "SMA crossover alerts",
			"conditions":     []string{"up", "down"},
			"example_target": 20.0,
		},
	}

	timeframes := []string{"1m", "5m", "15m", "1h", "4h", "1d"}

	notificationChannels := []string{"app", "email", "push", "sms"}

	c.JSON(http.StatusOK, gin.H{
		"alert_types":           alertTypes,
		"supported_timeframes":  timeframes,
		"notification_channels": notificationChannels,
	})
}
