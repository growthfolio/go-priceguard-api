package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/growthfolio/go-priceguard-api/internal/application/services"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	UserContextKey      = "user"
	UserIDContextKey    = "user_id"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	authService *services.AuthService
	logger      *logrus.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authService *services.AuthService, logger *logrus.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
		logger:      logger,
	}
}

// RequireAuth middleware that requires authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		token, err := m.extractTokenFromHeader(c)
		if err != nil {
			m.logger.WithError(err).Debug("Failed to extract token from header")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		user, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			m.logger.WithError(err).Debug("Token validation failed")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Add user to context
		c.Set(UserContextKey, user)
		c.Set(UserIDContextKey, user.ID)

		// Add user to request context for use in other layers
		ctx := context.WithValue(c.Request.Context(), UserContextKey, user)
		ctx = context.WithValue(ctx, UserIDContextKey, user.ID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	})
}

// OptionalAuth middleware that optionally authenticates users
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		token, err := m.extractTokenFromHeader(c)
		if err != nil {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		user, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			// Invalid token, continue without authentication
			m.logger.WithError(err).Debug("Optional auth: token validation failed")
			c.Next()
			return
		}

		// Add user to context
		c.Set(UserContextKey, user)
		c.Set(UserIDContextKey, user.ID)

		// Add user to request context
		ctx := context.WithValue(c.Request.Context(), UserContextKey, user)
		ctx = context.WithValue(ctx, UserIDContextKey, user.ID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	})
}

// WebSocketAuth authenticates WebSocket connections via query parameter
func (m *AuthMiddleware) WebSocketAuth() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Get token from query parameter for WebSocket connections
		token := c.Query("token")
		if token == "" {
			m.logger.Debug("WebSocket auth: no token provided")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication token required for WebSocket connection",
			})
			c.Abort()
			return
		}

		user, err := m.authService.ValidateToken(c.Request.Context(), token)
		if err != nil {
			m.logger.WithError(err).Debug("WebSocket auth: token validation failed")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Add user to context
		c.Set(UserContextKey, user)
		c.Set(UserIDContextKey, user.ID)

		// Add user to request context
		ctx := context.WithValue(c.Request.Context(), UserContextKey, user)
		ctx = context.WithValue(ctx, UserIDContextKey, user.ID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	})
}

// extractTokenFromHeader extracts the JWT token from the Authorization header
func (m *AuthMiddleware) extractTokenFromHeader(c *gin.Context) (string, error) {
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	if !strings.HasPrefix(authHeader, BearerPrefix) {
		return "", fmt.Errorf("authorization header must start with Bearer")
	}

	token := strings.TrimPrefix(authHeader, BearerPrefix)
	if token == "" {
		return "", fmt.Errorf("token is missing")
	}

	return token, nil
}

// GetUserFromContext extracts the authenticated user from the Gin context
func GetUserFromContext(c *gin.Context) (*entities.User, bool) {
	user, exists := c.Get(UserContextKey)
	if !exists {
		return nil, false
	}

	authUser, ok := user.(*entities.User)
	return authUser, ok
}

// GetUserIDFromContext extracts the authenticated user ID from the Gin context
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get(UserIDContextKey)
	if !exists {
		return uuid.Nil, false
	}

	id, ok := userID.(uuid.UUID)
	return id, ok
}

// MustGetUserFromContext extracts the user from context or panics
func MustGetUserFromContext(c *gin.Context) *entities.User {
	user, ok := GetUserFromContext(c)
	if !ok {
		panic("user not found in context - make sure RequireAuth middleware is used")
	}
	return user
}

// MustGetUserIDFromContext extracts the user ID from context or panics
func MustGetUserIDFromContext(c *gin.Context) uuid.UUID {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		panic("user ID not found in context - make sure RequireAuth middleware is used")
	}
	return userID
}
