package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
)

type UserHandler struct {
	userRepo         repositories.UserRepository
	userSettingsRepo repositories.UserSettingsRepository
}

// NewUserHandler creates a new user handler
func NewUserHandler(userRepo repositories.UserRepository, userSettingsRepo repositories.UserSettingsRepository) *UserHandler {
	return &UserHandler{
		userRepo:         userRepo,
		userSettingsRepo: userSettingsRepo,
	}
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get the authenticated user's profile information
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.User
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update the authenticated user's profile information
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body entities.User true "User profile data"
// @Success 200 {object} entities.User
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/user/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get existing user
	user, err := h.userRepo.GetByID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Parse update data
	var updateData struct {
		Name    *string `json:"name,omitempty"`
		Picture *string `json:"picture,omitempty"`
		Avatar  *string `json:"avatar,omitempty"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Update fields if provided
	if updateData.Name != nil {
		user.Name = *updateData.Name
	}
	if updateData.Picture != nil {
		user.Picture = updateData.Picture
	}
	if updateData.Avatar != nil {
		user.Avatar = updateData.Avatar
	}

	// Save updates
	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetSettings godoc
// @Summary Get user settings
// @Description Get the authenticated user's settings
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entities.UserSettings
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Settings not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/user/settings [get]
func (h *UserHandler) GetSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	settings, err := h.userSettingsRepo.GetByUserID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings godoc
// @Summary Update user settings
// @Description Update the authenticated user's settings
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param settings body entities.UserSettings true "User settings data"
// @Success 200 {object} entities.UserSettings
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Settings not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/user/settings [put]
func (h *UserHandler) UpdateSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get existing settings
	settings, err := h.userSettingsRepo.GetByUserID(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	// Parse update data
	var updateData struct {
		Theme              *string  `json:"theme,omitempty"`
		DefaultTimeframe   *string  `json:"default_timeframe,omitempty"`
		DefaultView        *string  `json:"default_view,omitempty"`
		NotificationsEmail *bool    `json:"notifications_email,omitempty"`
		NotificationsPush  *bool    `json:"notifications_push,omitempty"`
		NotificationsSMS   *bool    `json:"notifications_sms,omitempty"`
		RiskProfile        *string  `json:"risk_profile,omitempty"`
		FavoriteSymbols    []string `json:"favorite_symbols,omitempty"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Update fields if provided
	if updateData.Theme != nil {
		settings.Theme = *updateData.Theme
	}
	if updateData.DefaultTimeframe != nil {
		settings.DefaultTimeframe = *updateData.DefaultTimeframe
	}
	if updateData.DefaultView != nil {
		settings.DefaultView = *updateData.DefaultView
	}
	if updateData.NotificationsEmail != nil {
		settings.NotificationsEmail = *updateData.NotificationsEmail
	}
	if updateData.NotificationsPush != nil {
		settings.NotificationsPush = *updateData.NotificationsPush
	}
	if updateData.NotificationsSMS != nil {
		settings.NotificationsSMS = *updateData.NotificationsSMS
	}
	if updateData.RiskProfile != nil {
		settings.RiskProfile = *updateData.RiskProfile
	}
	if updateData.FavoriteSymbols != nil {
		settings.FavoriteSymbols = updateData.FavoriteSymbols
	}

	// Save updates
	if err := h.userSettingsRepo.Update(c.Request.Context(), settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}
