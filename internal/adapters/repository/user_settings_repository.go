package repository

import (
	"context"
	"fmt"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserSettingsRepositoryImpl implements the UserSettingsRepository interface
type UserSettingsRepositoryImpl struct {
	db *gorm.DB
}

// NewUserSettingsRepository creates a new user settings repository
func NewUserSettingsRepository(db *gorm.DB) repositories.UserSettingsRepository {
	return &UserSettingsRepositoryImpl{
		db: db,
	}
}

// Create creates new user settings
func (r *UserSettingsRepositoryImpl) Create(ctx context.Context, settings *entities.UserSettings) error {
	if err := r.db.WithContext(ctx).Create(settings).Error; err != nil {
		return fmt.Errorf("failed to create user settings: %w", err)
	}
	return nil
}

// GetByUserID retrieves user settings by user ID
func (r *UserSettingsRepositoryImpl) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSettings, error) {
	var settings entities.UserSettings
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&settings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user settings not found")
		}
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}
	return &settings, nil
}

// Update updates existing user settings
func (r *UserSettingsRepositoryImpl) Update(ctx context.Context, settings *entities.UserSettings) error {
	if err := r.db.WithContext(ctx).Save(settings).Error; err != nil {
		return fmt.Errorf("failed to update user settings: %w", err)
	}
	return nil
}

// Delete deletes user settings by user ID
func (r *UserSettingsRepositoryImpl) Delete(ctx context.Context, userID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&entities.UserSettings{}, "user_id = ?", userID).Error; err != nil {
		return fmt.Errorf("failed to delete user settings: %w", err)
	}
	return nil
}
