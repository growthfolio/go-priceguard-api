package entities

import (
	"time"
	
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	GoogleID  string    `json:"google_id" gorm:"uniqueIndex;not null"`
	Email     string    `json:"email" gorm:"uniqueIndex;not null"`
	Name      string    `json:"name" gorm:"not null"`
	Picture   *string   `json:"picture,omitempty"`
	Avatar    *string   `json:"avatar,omitempty"`
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	Settings      *UserSettings   `json:"settings,omitempty" gorm:"foreignKey:UserID"`
	Alerts        []Alert         `json:"alerts,omitempty" gorm:"foreignKey:UserID"`
	Notifications []Notification  `json:"notifications,omitempty" gorm:"foreignKey:UserID"`
	Sessions      []Session       `json:"sessions,omitempty" gorm:"foreignKey:UserID"`
}

// UserSettings represents user preferences and settings
type UserSettings struct {
	ID                 uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID             uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;uniqueIndex"`
	Theme              string         `json:"theme" gorm:"default:'dark'"`
	DefaultTimeframe   string         `json:"default_timeframe" gorm:"default:'1h'"`
	DefaultView        string         `json:"default_view" gorm:"default:'overview'"`
	NotificationsEmail bool           `json:"notifications_email" gorm:"default:true"`
	NotificationsPush  bool           `json:"notifications_push" gorm:"default:true"`
	NotificationsSMS   bool           `json:"notifications_sms" gorm:"default:false"`
	RiskProfile        string         `json:"risk_profile" gorm:"default:'moderate'"`
	FavoriteSymbols    pq.StringArray `json:"favorite_symbols" gorm:"type:text[]"`
	CreatedAt          time.Time      `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt          time.Time      `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// CryptoCurrency represents a cryptocurrency
type CryptoCurrency struct {
	ID         int       `json:"id" gorm:"primary_key;autoIncrement"`
	Symbol     string    `json:"symbol" gorm:"uniqueIndex;not null"`
	Name       string    `json:"name" gorm:"not null"`
	MarketType string    `json:"market_type" gorm:"default:'Spot'"`
	ImageURL   *string   `json:"image_url,omitempty"`
	Active     bool      `json:"active" gorm:"default:true"`
	CreatedAt  time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	Alerts         []Alert              `json:"alerts,omitempty" gorm:"foreignKey:Symbol;references:Symbol"`
	PriceHistory   []PriceHistory       `json:"price_history,omitempty" gorm:"foreignKey:Symbol;references:Symbol"`
	TechIndicators []TechnicalIndicator `json:"tech_indicators,omitempty" gorm:"foreignKey:Symbol;references:Symbol"`
}

// Alert represents a user alert
type Alert struct {
	ID            uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID        uuid.UUID      `json:"user_id" gorm:"type:uuid;not null;index"`
	Symbol        string         `json:"symbol" gorm:"not null;index"`
	AlertType     string         `json:"alert_type" gorm:"not null"` // 'price', 'rsi', 'ema_cross', etc.
	ConditionType string         `json:"condition_type" gorm:"not null"` // 'above', 'below', 'crosses'
	TargetValue   float64        `json:"target_value" gorm:"type:decimal(20,8);not null"`
	Timeframe     string         `json:"timeframe" gorm:"not null"`
	Enabled       bool           `json:"enabled" gorm:"default:true"`
	NotifyVia     pq.StringArray `json:"notify_via" gorm:"type:text[];default:'{app}'"`
	TriggeredAt   *time.Time     `json:"triggered_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time      `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	User          User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Notifications []Notification `json:"notifications,omitempty" gorm:"foreignKey:AlertID"`
}

// Notification represents a user notification
type Notification struct {
	ID               uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID           uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	AlertID          *uuid.UUID `json:"alert_id,omitempty" gorm:"type:uuid;index"`
	Title            string     `json:"title" gorm:"not null"`
	Message          string     `json:"message" gorm:"not null"`
	NotificationType string     `json:"notification_type" gorm:"not null"` // 'alert_triggered', 'system', etc.
	ReadAt           *time.Time `json:"read_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	User  User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Alert *Alert `json:"alert,omitempty" gorm:"foreignKey:AlertID"`
}

// PriceHistory represents historical price data
type PriceHistory struct {
	ID         int64     `json:"id" gorm:"primary_key;autoIncrement"`
	Symbol     string    `json:"symbol" gorm:"not null;index:idx_symbol_timeframe"`
	Timeframe  string    `json:"timeframe" gorm:"not null;index:idx_symbol_timeframe"`
	OpenPrice  float64   `json:"open_price" gorm:"type:decimal(20,8);not null"`
	HighPrice  float64   `json:"high_price" gorm:"type:decimal(20,8);not null"`
	LowPrice   float64   `json:"low_price" gorm:"type:decimal(20,8);not null"`
	ClosePrice float64   `json:"close_price" gorm:"type:decimal(20,8);not null"`
	Volume     float64   `json:"volume" gorm:"type:decimal(30,8);not null"`
	Timestamp  time.Time `json:"timestamp" gorm:"not null;index;uniqueIndex:idx_unique_price"`
	CreatedAt  time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// TechnicalIndicator represents cached technical indicators
type TechnicalIndicator struct {
	ID            int64                  `json:"id" gorm:"primary_key;autoIncrement"`
	Symbol        string                 `json:"symbol" gorm:"not null;index:idx_symbol_timeframe_type"`
	Timeframe     string                 `json:"timeframe" gorm:"not null;index:idx_symbol_timeframe_type"`
	IndicatorType string                 `json:"indicator_type" gorm:"not null;index:idx_symbol_timeframe_type"` // 'rsi', 'ema', 'sma', 'supertrend', etc.
	Value         *float64               `json:"value,omitempty" gorm:"type:decimal(20,8)"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" gorm:"type:jsonb"`
	Timestamp     time.Time              `json:"timestamp" gorm:"not null;index;uniqueIndex:idx_unique_indicator"`
	CreatedAt     time.Time              `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// Session represents a user session
type Session struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	TokenHash string    `json:"token_hash" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`

	// Relationships
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// RawCryptoData represents the real-time crypto data structure expected by the frontend
type RawCryptoData struct {
	DashboardData struct {
		ID         int    `json:"id"`
		Symbol     string `json:"symbol"`
		MarketType string `json:"marketType"`
		ImageURL   string `json:"imgurl"`
		Active     bool   `json:"active"`
	} `json:"dashboardData"`
	
	// Price changes
	PriceChange1m  string `json:"priceChange_1m"`
	PriceChange5m  string `json:"priceChange_5m"`
	PriceChange15m string `json:"priceChange_15m"`
	PriceChange1h  string `json:"priceChange_1h"`
	PriceChange1d  string `json:"priceChange_1d"`
	
	// Pullback entries
	PullbackEntry5m  *string `json:"pullbackEntry_5m"`
	PullbackEntry15m *string `json:"pullbackEntry_15m"`
	PullbackEntry1h  *string `json:"pullbackEntry_1h"`
	PullbackEntry4h  *string `json:"pullbackEntry_4h"`
	PullbackEntry1d  *string `json:"pullbackEntry_1d"`
	
	// SuperTrend
	SuperTrend4h5m  *string `json:"superTrend4h_5m"`
	SuperTrend4h15m *string `json:"superTrend4h_15m"`
	SuperTrend4h1h  *string `json:"superTrend4h_1h"`
	
	// True Range
	TrueRange1m  string `json:"trueRange_1m"`
	TrueRange5m  string `json:"trueRange_5m"`
	TrueRange15m string `json:"trueRange_15m"`
	TrueRange1h  string `json:"trueRange_1h"`
	
	// RSI (formato especial: "45.32<[50]")
	RSI5m  string `json:"rsi_5m"`
	RSI15m string `json:"rsi_15m"`
	RSI1h  string `json:"rsi_1h"`
	RSI4h  string `json:"rsi_4h"`
	RSI1d  string `json:"rsi_1d"`
	
	// EMA Trends
	EMATrend15m string `json:"ematrend_15m"`
	EMATrend1h  string `json:"ematrend_1h"`
	EMATrend4h  string `json:"ematrend_4h"`
	EMATrend1d  string `json:"ematrend_1d"`
}
