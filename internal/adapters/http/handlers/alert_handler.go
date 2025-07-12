package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/repositories"
)

type AlertHandler struct {
	alertRepo repositories.AlertRepository
}

// NewAlertHandler creates a new alert handler
func NewAlertHandler(alertRepo repositories.AlertRepository) *AlertHandler {
	return &AlertHandler{
		alertRepo: alertRepo,
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
