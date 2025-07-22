package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/growthfolio/go-priceguard-api/internal/adapters/http/middleware"
	"github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/sirupsen/logrus"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *services.AuthService
	logger      *logrus.Logger
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *services.AuthService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

// RefreshTokenRequest represents the refresh token request payload
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest represents the logout request payload
type LogoutRequest struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Login handles Google OAuth login
// @Summary Login with Google OAuth
// @Description Authenticate user with Google ID token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} services.LoginResult
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid login request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request format",
		})
		return
	}

	result, err := h.authService.LoginWithGoogleIDToken(c.Request.Context(), req.IDToken)
	if err != nil {
		if strings.Contains(err.Error(), "failed to create user") {
			h.logger.WithError(err).Error("Failed to create user during Google login")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": "Failed to create user",
			})
			return
		}
		h.logger.WithError(err).Error("Login failed")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication failed",
		})
		return
	}

	h.logger.WithField("user_id", result.User.ID).Info("User logged in via API")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Generate new access token from refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} services.AuthTokens
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid refresh token request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request format",
		})
		return
	}

	tokens, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.logger.WithError(err).Error("Token refresh failed")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Failed to refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    tokens,
	})
}

// Logout handles user logout
// @Summary Logout user
// @Description Logout user and invalidate tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LogoutRequest true "Logout request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid logout request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Invalid request format",
		})
		return
	}

	// If access token is not provided in request body, try to get it from header
	accessToken := req.AccessToken
	if accessToken == "" {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 {
			accessToken = authHeader[7:] // Remove "Bearer " prefix
		}
	}

	if accessToken == "" {
		h.logger.Error("No access token provided for logout")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "bad_request",
			"message": "Access token is required",
		})
		return
	}

	err := h.authService.Logout(c.Request.Context(), accessToken, req.RefreshToken)
	if err != nil {
		h.logger.WithError(err).Error("Logout failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Logout failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

// VerifyToken verifies the current token and returns user info
// @Summary Verify token
// @Description Verify current access token and return user information
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /auth/verify [get]
func (h *AuthHandler) VerifyToken(c *gin.Context) {
	user := middleware.MustGetUserFromContext(c)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"valid": true,
			"user":  user,
		},
	})
}

// RegisterRoutes registers authentication routes
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.Logout)
		auth.GET("/verify", authMiddleware.RequireAuth(), h.VerifyToken)
	}
}
