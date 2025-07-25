package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
)

type priceHistoryRepository struct {
	db *gorm.DB
}

// NewPriceHistoryRepository creates a new price history repository
func NewPriceHistoryRepository(db *gorm.DB) repositories.PriceHistoryRepository {
	return &priceHistoryRepository{
		db: db,
	}
}

func (r *priceHistoryRepository) Create(ctx context.Context, history *entities.PriceHistory) error {
	history.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *priceHistoryRepository) GetBySymbol(ctx context.Context, symbol, timeframe string, limit int) ([]entities.PriceHistory, error) {
	var histories []entities.PriceHistory
	query := r.db.WithContext(ctx).Where("symbol = ? AND timeframe = ?", symbol, timeframe).Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&histories).Error
	return histories, err
}

func (r *priceHistoryRepository) GetLatest(ctx context.Context, symbol, timeframe string) (*entities.PriceHistory, error) {
	var history entities.PriceHistory
	err := r.db.WithContext(ctx).
		Where("symbol = ? AND timeframe = ?", symbol, timeframe).
		Order("timestamp DESC").
		First(&history).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

func (r *priceHistoryRepository) BulkInsert(ctx context.Context, histories []entities.PriceHistory) error {
	if len(histories) == 0 {
		return nil
	}

	// Set created_at for all records
	now := time.Now()
	for i := range histories {
		histories[i].CreatedAt = now
	}

	return r.db.WithContext(ctx).CreateInBatches(histories, 1000).Error
}

func (r *priceHistoryRepository) DeleteOld(ctx context.Context, symbol, timeframe string, keepDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -keepDays)
	return r.db.WithContext(ctx).
		Where("symbol = ? AND timeframe = ? AND timestamp < ?", symbol, timeframe, cutoffDate).
		Delete(&entities.PriceHistory{}).Error
}
