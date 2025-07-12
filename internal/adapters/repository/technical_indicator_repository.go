package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/repositories"
)

type technicalIndicatorRepository struct {
	db *gorm.DB
}

// NewTechnicalIndicatorRepository creates a new technical indicator repository
func NewTechnicalIndicatorRepository(db *gorm.DB) repositories.TechnicalIndicatorRepository {
	return &technicalIndicatorRepository{
		db: db,
	}
}

func (r *technicalIndicatorRepository) Create(ctx context.Context, indicator *entities.TechnicalIndicator) error {
	indicator.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(indicator).Error
}

func (r *technicalIndicatorRepository) GetBySymbol(ctx context.Context, symbol, timeframe, indicatorType string, limit int) ([]entities.TechnicalIndicator, error) {
	var indicators []entities.TechnicalIndicator
	query := r.db.WithContext(ctx).
		Where("symbol = ? AND timeframe = ? AND indicator_type = ?", symbol, timeframe, indicatorType).
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&indicators).Error
	return indicators, err
}

func (r *technicalIndicatorRepository) GetLatest(ctx context.Context, symbol, timeframe, indicatorType string) (*entities.TechnicalIndicator, error) {
	var indicator entities.TechnicalIndicator
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND timeframe = ? AND indicator_type = ?", symbol, timeframe, indicatorType).
		Order("timestamp DESC").
		First(&indicator).Error
	if err != nil {
		return nil, err
	}
	return &indicator, nil
}

func (r *technicalIndicatorRepository) BulkInsert(ctx context.Context, indicators []entities.TechnicalIndicator) error {
	if len(indicators) == 0 {
		return nil
	}

	// Set created_at for all records
	now := time.Now()
	for i := range indicators {
		indicators[i].CreatedAt = now
	}

	return r.db.WithContext(ctx).CreateInBatches(indicators, 1000).Error
}

func (r *technicalIndicatorRepository) DeleteOld(ctx context.Context, symbol, timeframe string, keepDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -keepDays)
	return r.db.WithContext(ctx).
		Where("symbol = ? AND timeframe = ? AND timestamp < ?", symbol, timeframe, cutoffDate).
		Delete(&entities.TechnicalIndicator{}).Error
}
