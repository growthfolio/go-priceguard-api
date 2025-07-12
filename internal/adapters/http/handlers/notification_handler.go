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

type NotificationHandler struct {
	notificationRepo    repositories.NotificationRepository
	notificationService *services.NotificationService
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(
	notificationRepo repositories.NotificationRepository,
	notificationService *services.NotificationService,
) *NotificationHandler {
	return &NotificationHandler{
		notificationRepo:    notificationRepo,
		notificationService: notificationService,
	}
}

// GetNotifications godoc
// @Summary Get user notifications
// @Description Get list of notifications for the authenticated user
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Param unread_only query bool false "Show only unread notifications" default(false)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/notifications [get]
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	unreadOnlyStr := c.DefaultQuery("unread_only", "false")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	unreadOnly, _ := strconv.ParseBool(unreadOnlyStr)

	// Validate limits
	if limit > 100 {
		limit = 100
	}
	if limit <= 0 {
		limit = 50
	}

	var notifications []entities.Notification
	var err error

	if unreadOnly {
		notifications, err = h.notificationRepo.GetUnread(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	} else {
		notifications, err = h.notificationRepo.GetByUserID(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        notifications,
		"limit":       limit,
		"offset":      offset,
		"count":       len(notifications),
		"unread_only": unreadOnly,
	})
}

// MarkAsRead godoc
// @Summary Mark notifications as read
// @Description Mark one or more notifications as read for the authenticated user
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data body map[string][]string true "Notification IDs to mark as read"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/notifications/mark-read [post]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var requestData struct {
		NotificationIDs []string `json:"notification_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	if len(requestData.NotificationIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No notification IDs provided"})
		return
	}

	// Parse UUIDs
	var notificationIDs []uuid.UUID
	for _, idStr := range requestData.NotificationIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID format"})
			return
		}
		notificationIDs = append(notificationIDs, id)
	}

	// Mark as read
	if err := h.notificationRepo.MarkAsRead(c.Request.Context(), notificationIDs, userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notifications marked as read",
		"count":   len(notificationIDs),
	})
}

// CreateTestNotification godoc
// @Summary Create test notification
// @Description Create a test notification for the authenticated user (development only)
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]interface{} true "Test notification data"
// @Success 201 {object} entities.Notification
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/notifications/test [post]
func (h *NotificationHandler) CreateTestNotification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var request struct {
		Title    string                 `json:"title" binding:"required"`
		Message  string                 `json:"message" binding:"required"`
		Type     string                 `json:"type"`
		Channels []string               `json:"channels"`
		Data     map[string]interface{} `json:"data,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Set defaults
	if request.Type == "" {
		request.Type = "test"
	}

	// Create in-app notification
	notification, err := h.notificationService.CreateNotification(
		c.Request.Context(),
		userID.(uuid.UUID),
		request.Type,
		request.Title,
		request.Message,
		request.Data,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
		return
	}

	// Queue for other channels if specified
	if len(request.Channels) > 0 {
		channels := make([]services.NotificationChannel, 0, len(request.Channels))
		for _, ch := range request.Channels {
			switch ch {
			case "email":
				channels = append(channels, services.ChannelEmail)
			case "push":
				channels = append(channels, services.ChannelPush)
			case "sms":
				channels = append(channels, services.ChannelSMS)
			}
		}

		if len(channels) > 0 {
			queuedNotification := &services.QueuedNotification{
				UserID:   userID.(uuid.UUID),
				Type:     request.Type,
				Title:    request.Title,
				Message:  request.Message,
				Channels: channels,
				Priority: services.PriorityNormal,
				Data:     request.Data,
			}

			err = h.notificationService.QueueNotification(c.Request.Context(), queuedNotification)
			if err != nil {
				// Log error but don't fail the request
				c.Header("X-Queue-Warning", "Failed to queue for external channels")
			}
		}
	}

	c.JSON(http.StatusCreated, notification)
}

// GetNotificationStats godoc
// @Summary Get notification statistics
// @Description Get statistics about the notification system
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/notifications/stats [get]
func (h *NotificationHandler) GetNotificationStats(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	stats, err := h.notificationService.GetNotificationStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get notification stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// DeleteNotification godoc
// @Summary Delete notification
// @Description Delete a notification for the authenticated user
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Notification ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Notification not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/notifications/{id} [delete]
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	notificationIDStr := c.Param("id")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	// Get notification to check ownership
	notification, err := h.notificationRepo.GetByID(c.Request.Context(), notificationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	// Check if user owns the notification
	if notification.UserID != userID.(uuid.UUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if err := h.notificationRepo.Delete(c.Request.Context(), notificationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		return
	}

	c.Status(http.StatusNoContent)
}

// MarkAllAsRead godoc
// @Summary Mark all notifications as read
// @Description Mark all notifications as read for the authenticated user
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/notifications/mark-all-read [post]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	count, err := h.notificationRepo.MarkAllAsReadByUserID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "All notifications marked as read",
		"updated_count": count,
		"marked_at":     time.Now(),
	})
}
