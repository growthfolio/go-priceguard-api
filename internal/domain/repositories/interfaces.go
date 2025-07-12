package repositories

import (
	"context"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/google/uuid"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *entities.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
	GetByEmail(ctx context.Context, email string) (*entities.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*entities.User, error)
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserSettingsRepository defines the interface for user settings operations
type UserSettingsRepository interface {
	Create(ctx context.Context, settings *entities.UserSettings) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSettings, error)
	Update(ctx context.Context, settings *entities.UserSettings) error
	Delete(ctx context.Context, userID uuid.UUID) error
}

// CryptoCurrencyRepository defines the interface for cryptocurrency operations
type CryptoCurrencyRepository interface {
	Create(ctx context.Context, crypto *entities.CryptoCurrency) error
	GetByID(ctx context.Context, id int) (*entities.CryptoCurrency, error)
	GetBySymbol(ctx context.Context, symbol string) (*entities.CryptoCurrency, error)
	GetAll(ctx context.Context, limit, offset int) ([]entities.CryptoCurrency, error)
	GetActive(ctx context.Context, limit, offset int) ([]entities.CryptoCurrency, error)
	Update(ctx context.Context, crypto *entities.CryptoCurrency) error
	Delete(ctx context.Context, id int) error
}

// AlertRepository defines the interface for alert operations
type AlertRepository interface {
	Create(ctx context.Context, alert *entities.Alert) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Alert, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Alert, error)
	GetBySymbol(ctx context.Context, symbol string) ([]entities.Alert, error)
	GetEnabled(ctx context.Context) ([]entities.Alert, error)
	Update(ctx context.Context, alert *entities.Alert) error
	Delete(ctx context.Context, id uuid.UUID) error
	MarkTriggered(ctx context.Context, id uuid.UUID) error
}

// NotificationRepository defines the interface for notification operations
type NotificationRepository interface {
	Create(ctx context.Context, notification *entities.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.Notification, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Notification, error)
	GetUnread(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entities.Notification, error)
	MarkAsRead(ctx context.Context, ids []uuid.UUID, userID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PriceHistoryRepository defines the interface for price history operations
type PriceHistoryRepository interface {
	Create(ctx context.Context, history *entities.PriceHistory) error
	GetBySymbol(ctx context.Context, symbol, timeframe string, limit int) ([]entities.PriceHistory, error)
	GetLatest(ctx context.Context, symbol, timeframe string) (*entities.PriceHistory, error)
	BulkInsert(ctx context.Context, histories []entities.PriceHistory) error
	DeleteOld(ctx context.Context, symbol, timeframe string, keepDays int) error
}

// TechnicalIndicatorRepository defines the interface for technical indicator operations
type TechnicalIndicatorRepository interface {
	Create(ctx context.Context, indicator *entities.TechnicalIndicator) error
	GetBySymbol(ctx context.Context, symbol, timeframe, indicatorType string, limit int) ([]entities.TechnicalIndicator, error)
	GetLatest(ctx context.Context, symbol, timeframe, indicatorType string) (*entities.TechnicalIndicator, error)
	BulkInsert(ctx context.Context, indicators []entities.TechnicalIndicator) error
	DeleteOld(ctx context.Context, symbol, timeframe string, keepDays int) error
}

// SessionRepository defines the interface for session operations
type SessionRepository interface {
	Create(ctx context.Context, session *entities.Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*entities.Session, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]entities.Session, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteExpired(ctx context.Context) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}
