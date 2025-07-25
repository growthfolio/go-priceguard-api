package repository

import (
	"context"
	"fmt"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
	"gorm.io/gorm"
)

// CryptoCurrencyRepositoryImpl implements the CryptoCurrencyRepository interface
type CryptoCurrencyRepositoryImpl struct {
	db *gorm.DB
}

// NewCryptoCurrencyRepository creates a new cryptocurrency repository
func NewCryptoCurrencyRepository(db *gorm.DB) repositories.CryptoCurrencyRepository {
	return &CryptoCurrencyRepositoryImpl{
		db: db,
	}
}

// Create creates a new cryptocurrency
func (r *CryptoCurrencyRepositoryImpl) Create(ctx context.Context, crypto *entities.CryptoCurrency) error {
	if err := r.db.WithContext(ctx).Create(crypto).Error; err != nil {
		return fmt.Errorf("failed to create cryptocurrency: %w", err)
	}
	return nil
}

// GetByID retrieves a cryptocurrency by ID
func (r *CryptoCurrencyRepositoryImpl) GetByID(ctx context.Context, id int) (*entities.CryptoCurrency, error) {
	var crypto entities.CryptoCurrency
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&crypto).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cryptocurrency not found")
		}
		return nil, fmt.Errorf("failed to get cryptocurrency by ID: %w", err)
	}
	return &crypto, nil
}

// GetBySymbol retrieves a cryptocurrency by symbol
func (r *CryptoCurrencyRepositoryImpl) GetBySymbol(ctx context.Context, symbol string) (*entities.CryptoCurrency, error) {
	var crypto entities.CryptoCurrency
	if err := r.db.WithContext(ctx).Where("symbol = ?", symbol).First(&crypto).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cryptocurrency not found")
		}
		return nil, fmt.Errorf("failed to get cryptocurrency by symbol: %w", err)
	}
	return &crypto, nil
}

// GetAll retrieves all cryptocurrencies with pagination
func (r *CryptoCurrencyRepositoryImpl) GetAll(ctx context.Context, limit, offset int) ([]entities.CryptoCurrency, error) {
	var cryptos []entities.CryptoCurrency
	query := r.db.WithContext(ctx).Order("name ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&cryptos).Error; err != nil {
		return nil, fmt.Errorf("failed to get cryptocurrencies: %w", err)
	}
	return cryptos, nil
}

// GetActive retrieves all active cryptocurrencies with pagination
func (r *CryptoCurrencyRepositoryImpl) GetActive(ctx context.Context, limit, offset int) ([]entities.CryptoCurrency, error) {
	var cryptos []entities.CryptoCurrency
	query := r.db.WithContext(ctx).Where("active = ?", true).Order("name ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&cryptos).Error; err != nil {
		return nil, fmt.Errorf("failed to get active cryptocurrencies: %w", err)
	}
	return cryptos, nil
}

// Update updates an existing cryptocurrency
func (r *CryptoCurrencyRepositoryImpl) Update(ctx context.Context, crypto *entities.CryptoCurrency) error {
	if err := r.db.WithContext(ctx).Save(crypto).Error; err != nil {
		return fmt.Errorf("failed to update cryptocurrency: %w", err)
	}
	return nil
}

// Delete deletes a cryptocurrency by ID
func (r *CryptoCurrencyRepositoryImpl) Delete(ctx context.Context, id int) error {
	if err := r.db.WithContext(ctx).Delete(&entities.CryptoCurrency{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete cryptocurrency: %w", err)
	}
	return nil
}
