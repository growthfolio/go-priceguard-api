package services

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	GoogleID string    `json:"google_id"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations
type JWTService struct {
	secretKey         []byte
	expiration        time.Duration
	refreshExpiration time.Duration
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey string, expiration, refreshExpiration time.Duration) *JWTService {
	return &JWTService{
		secretKey:         []byte(secretKey),
		expiration:        expiration,
		refreshExpiration: refreshExpiration,
	}
}

// GenerateTokens generates both access and refresh tokens
func (j *JWTService) GenerateTokens(userID uuid.UUID, email, name, googleID string) (accessToken, refreshToken string, err error) {
	// Generate access token
	accessToken, err = j.generateToken(userID, email, name, googleID, j.expiration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token with longer expiration
	refreshToken, err = j.generateToken(userID, email, name, googleID, j.refreshExpiration)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// generateToken creates a JWT token with the given expiration
func (j *JWTService) generateToken(userID uuid.UUID, email, name, googleID string, expiration time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:   userID,
		Email:    email,
		Name:     name,
		GoogleID: googleID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "priceguard-api",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateToken validates a JWT token and returns the claims
func (j *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshToken generates a new access token from a valid refresh token
func (j *JWTService) RefreshToken(refreshTokenString string) (newAccessToken string, err error) {
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new access token
	newAccessToken, err = j.generateToken(claims.UserID, claims.Email, claims.Name, claims.GoogleID, j.expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate new access token: %w", err)
	}

	return newAccessToken, nil
}

// GetTokenHash generates a hash of the token for storage/blacklisting
func (j *JWTService) GetTokenHash(tokenString string) string {
	hash := sha256.Sum256([]byte(tokenString))
	return fmt.Sprintf("%x", hash)
}

// ExtractUserIDFromToken extracts user ID from token without full validation
func (j *JWTService) ExtractUserIDFromToken(tokenString string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok {
		return claims.UserID, nil
	}

	return uuid.Nil, fmt.Errorf("invalid token claims")
}

// GetTokenExpiration returns the expiration time of a token
func (j *JWTService) GetTokenExpiration(tokenString string) (time.Time, error) {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return time.Time{}, err
	}

	if claims.ExpiresAt == nil {
		return time.Time{}, fmt.Errorf("token has no expiration")
	}

	return claims.ExpiresAt.Time, nil
}
