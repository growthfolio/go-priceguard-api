package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/repositories"
)

type alertRepository struct {
	db *gorm.DB
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *gorm.DB) repositories.AlertRepository {
	return &alertRepository{
		db: db,
	}
}

func (r *alertRepository) Create(ctx context.Context, alert *entities.Alert) error {
	if alert.ID == uuid.Nil {
		alert.ID = uuid.New()
	}
	alert.CreatedAt = time.Now()
	alert.UpdatedAt = time.Now()

	return r.db.WithContext(ctx).Create(alert).Error
}

func (r *alertRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Alert, error) {
	var alert entities.Alert
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&alert).Error
	if err != nil {
		return nil, err
	}
	return &alert, nil
}

func (r *alertRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Alert, error) {
	var alerts []entities.Alert
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&alerts).Error
	return alerts, err
}

func (r *alertRepository) GetBySymbol(ctx context.Context, symbol string) ([]entities.Alert, error) {
	var alerts []entities.Alert
	err := r.db.WithContext(ctx).Where("symbol = ?", symbol).Find(&alerts).Error
	return alerts, err
}

func (r *alertRepository) GetEnabled(ctx context.Context) ([]entities.Alert, error) {
	var alerts []entities.Alert
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&alerts).Error
	return alerts, err
}

func (r *alertRepository) MarkTriggered(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&entities.Alert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_triggered": time.Now(),
			"updated_at":     time.Now(),
		}).Error
}

func (r *alertRepository) Update(ctx context.Context, alert *entities.Alert) error {
	alert.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(alert).Error
}

func (r *alertRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&entities.Alert{}, id).Error
}
