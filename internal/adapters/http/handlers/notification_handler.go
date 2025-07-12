package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/repositories"
)

type NotificationHandler struct {
	notificationRepo repositories.NotificationRepository
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(notificationRepo repositories.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{
		notificationRepo: notificationRepo,
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
