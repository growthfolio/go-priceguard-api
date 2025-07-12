package services

import (
	"context"
	"fmt"
	"time"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/repositories"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/services"
	"github.com/felipe-macedo/go-priceguard-api/internal/infrastructure/database"
	"github.com/sirupsen/logrus"
)

// AuthService handles authentication operations
type AuthService struct {
	userRepo      repositories.UserRepository
	sessionRepo   repositories.SessionRepository
	settingsRepo  repositories.UserSettingsRepository
	jwtService    *services.JWTService
	googleService *services.GoogleOAuthService
	redisClient   *database.RedisClient
	logger        *logrus.Logger
}

// AuthTokens represents the authentication tokens
type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// LoginResult represents the result of a login operation
type LoginResult struct {
	User   *entities.User `json:"user"`
	Tokens *AuthTokens    `json:"tokens"`
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo repositories.UserRepository,
	sessionRepo repositories.SessionRepository,
	settingsRepo repositories.UserSettingsRepository,
	jwtService *services.JWTService,
	googleService *services.GoogleOAuthService,
	redisClient *database.RedisClient,
	logger *logrus.Logger,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		sessionRepo:   sessionRepo,
		settingsRepo:  settingsRepo,
		jwtService:    jwtService,
		googleService: googleService,
		redisClient:   redisClient,
		logger:        logger,
	}
}

// LoginWithGoogleIDToken authenticates a user with Google ID token
func (a *AuthService) LoginWithGoogleIDToken(ctx context.Context, idToken string) (*LoginResult, error) {
	// Validate Google ID token
	googleUser, err := a.googleService.ValidateIDToken(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("invalid Google ID token: %w", err)
	}

	// Check if user exists
	user, err := a.userRepo.GetByGoogleID(ctx, googleUser.ID)
	if err != nil && err.Error() != "record not found" {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Create user if doesn't exist
	if user == nil {
		user = &entities.User{
			GoogleID: googleUser.ID,
			Email:    googleUser.Email,
			Name:     googleUser.Name,
			Picture:  &googleUser.Picture,
		}

		if err := a.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to create user: %w", err)
		}

		// Create default user settings
		settings := &entities.UserSettings{
			UserID:             user.ID,
			Theme:              "dark",
			DefaultTimeframe:   "1h",
			DefaultView:        "overview",
			NotificationsEmail: true,
			NotificationsPush:  true,
			NotificationsSMS:   false,
			RiskProfile:        "moderate",
			FavoriteSymbols:    []string{},
		}

		if err := a.settingsRepo.Create(ctx, settings); err != nil {
			a.logger.WithError(err).Error("Failed to create default user settings")
		}

		a.logger.WithField("user_id", user.ID).Info("New user created")
	} else {
		// Update user info from Google
		user.Name = googleUser.Name
		user.Picture = &googleUser.Picture
		if err := a.userRepo.Update(ctx, user); err != nil {
			a.logger.WithError(err).Error("Failed to update user info")
		}
	}

	// Generate JWT tokens
	accessToken, refreshToken, err := a.jwtService.GenerateTokens(user.ID, user.Email, user.Name, user.GoogleID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store session in database
	tokenHash := a.jwtService.GetTokenHash(refreshToken)
	session := &entities.Session{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	if err := a.sessionRepo.Create(ctx, session); err != nil {
		a.logger.WithError(err).Error("Failed to create session")
	}

	// Store session in Redis for quick lookup
	if err := a.redisClient.SetSession(ctx, session.ID.String(), user.ID.String(), 7*24*time.Hour); err != nil {
		a.logger.WithError(err).Error("Failed to cache session in Redis")
	}

	tokens := &AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(24 * time.Hour.Seconds()), // 24 hours for access token
		TokenType:    "Bearer",
	}

	a.logger.WithField("user_id", user.ID).Info("User logged in successfully")

	return &LoginResult{
		User:   user,
		Tokens: tokens,
	}, nil
}

// RefreshToken generates new tokens from a refresh token
func (a *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	// Validate refresh token
	claims, err := a.jwtService.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Check if token is blacklisted
	tokenHash := a.jwtService.GetTokenHash(refreshToken)
	isBlacklisted, err := a.redisClient.IsTokenBlacklisted(ctx, tokenHash)
	if err != nil {
		a.logger.WithError(err).Error("Failed to check token blacklist")
	}
	if isBlacklisted {
		return nil, fmt.Errorf("token is blacklisted")
	}

	// Verify session exists
	session, err := a.sessionRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if session is expired
	if session.ExpiresAt.Before(time.Now()) {
		a.sessionRepo.Delete(ctx, session.ID)
		return nil, fmt.Errorf("session expired")
	}

	// Get user
	user, err := a.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Generate new access token
	newAccessToken, err := a.jwtService.RefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	tokens := &AuthTokens{
		AccessToken:  newAccessToken,
		RefreshToken: refreshToken, // Keep the same refresh token
		ExpiresIn:    int64(24 * time.Hour.Seconds()),
		TokenType:    "Bearer",
	}

	a.logger.WithField("user_id", user.ID).Info("Token refreshed successfully")

	return tokens, nil
}

// Logout invalidates the user's session and blacklists the token
func (a *AuthService) Logout(ctx context.Context, accessToken, refreshToken string) error {
	// Get user ID from access token
	userID, err := a.jwtService.ExtractUserIDFromToken(accessToken)
	if err != nil {
		return fmt.Errorf("failed to extract user ID: %w", err)
	}

	// Blacklist both tokens
	accessTokenHash := a.jwtService.GetTokenHash(accessToken)
	refreshTokenHash := a.jwtService.GetTokenHash(refreshToken)

	// Get token expiration for blacklist TTL
	accessExpiration, _ := a.jwtService.GetTokenExpiration(accessToken)
	refreshExpiration, _ := a.jwtService.GetTokenExpiration(refreshToken)

	// Blacklist tokens in Redis
	if !accessExpiration.IsZero() {
		ttl := time.Until(accessExpiration)
		if ttl > 0 {
			a.redisClient.BlacklistToken(ctx, accessTokenHash, ttl)
		}
	}

	if !refreshExpiration.IsZero() {
		ttl := time.Until(refreshExpiration)
		if ttl > 0 {
			a.redisClient.BlacklistToken(ctx, refreshTokenHash, ttl)
		}
	}

	// Remove session from database
	if err := a.sessionRepo.DeleteByTokenHash(ctx, refreshTokenHash); err != nil {
		a.logger.WithError(err).Error("Failed to delete session from database")
	}

	// Remove session from Redis
	sessions, err := a.redisClient.GetWebSocketConnections(ctx, userID.String())
	if err == nil {
		for _, sessionID := range sessions {
			a.redisClient.DeleteSession(ctx, sessionID)
		}
	}

	a.logger.WithField("user_id", userID).Info("User logged out successfully")

	return nil
}

// ValidateToken validates an access token and returns user information
func (a *AuthService) ValidateToken(ctx context.Context, accessToken string) (*entities.User, error) {
	// Validate JWT token
	claims, err := a.jwtService.ValidateToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check if token is blacklisted
	tokenHash := a.jwtService.GetTokenHash(accessToken)
	isBlacklisted, err := a.redisClient.IsTokenBlacklisted(ctx, tokenHash)
	if err != nil {
		a.logger.WithError(err).Error("Failed to check token blacklist")
	}
	if isBlacklisted {
		return nil, fmt.Errorf("token is blacklisted")
	}

	// Get user from database
	user, err := a.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// CleanupExpiredSessions removes expired sessions
func (a *AuthService) CleanupExpiredSessions(ctx context.Context) error {
	return a.sessionRepo.DeleteExpired(ctx)
}
