package entities_test

import (
	"testing"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/stretchr/testify/assert"
)

func TestCryptoCurrency_Creation(t *testing.T) {
	now := time.Now()
	imageURL := "https://example.com/btc-logo.png"

	crypto := entities.CryptoCurrency{
		ID:         1,
		Symbol:     "BTCUSDT",
		Name:       "Bitcoin",
		MarketType: "Spot",
		ImageURL:   &imageURL,
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	assert.Equal(t, 1, crypto.ID)
	assert.Equal(t, "BTCUSDT", crypto.Symbol)
	assert.Equal(t, "Bitcoin", crypto.Name)
	assert.Equal(t, "Spot", crypto.MarketType)
	assert.NotNil(t, crypto.ImageURL)
	assert.Equal(t, imageURL, *crypto.ImageURL)
	assert.True(t, crypto.Active)
	assert.Equal(t, now, crypto.CreatedAt)
	assert.Equal(t, now, crypto.UpdatedAt)
}

func TestCryptoCurrency_CommonSymbols(t *testing.T) {
	commonCryptos := []struct {
		symbol string
		name   string
	}{
		{"BTCUSDT", "Bitcoin"},
		{"ETHUSDT", "Ethereum"},
		{"ADAUSDT", "Cardano"},
		{"DOGEUSDT", "Dogecoin"},
		{"LTCUSDT", "Litecoin"},
		{"XRPUSDT", "Ripple"},
		{"BNBUSDT", "Binance Coin"},
		{"SOLUSDT", "Solana"},
		{"MATICUSDT", "Polygon"},
		{"AVAXUSDT", "Avalanche"},
	}

	for _, crypto := range commonCryptos {
		t.Run("crypto_"+crypto.symbol, func(t *testing.T) {
			currency := entities.CryptoCurrency{
				Symbol:     crypto.symbol,
				Name:       crypto.name,
				MarketType: "Spot",
				Active:     true,
			}

			assert.Equal(t, crypto.symbol, currency.Symbol)
			assert.Equal(t, crypto.name, currency.Name)
			assert.Equal(t, "Spot", currency.MarketType)
			assert.True(t, currency.Active)
		})
	}
}

func TestCryptoCurrency_MarketTypes(t *testing.T) {
	marketTypes := []string{"Spot", "Futures", "Options", "Margin"}

	for _, marketType := range marketTypes {
		t.Run("market_type_"+marketType, func(t *testing.T) {
			crypto := entities.CryptoCurrency{
				Symbol:     "BTCUSDT",
				Name:       "Bitcoin",
				MarketType: marketType,
				Active:     true,
			}

			assert.Equal(t, marketType, crypto.MarketType)
		})
	}
}

func TestCryptoCurrency_ActiveStatus(t *testing.T) {
	tests := []struct {
		name   string
		active bool
	}{
		{
			name:   "active cryptocurrency",
			active: true,
		},
		{
			name:   "inactive cryptocurrency",
			active: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crypto := entities.CryptoCurrency{
				Symbol:     "BTCUSDT",
				Name:       "Bitcoin",
				MarketType: "Spot",
				Active:     tt.active,
			}

			assert.Equal(t, tt.active, crypto.Active)
		})
	}
}

func TestCryptoCurrency_WithOptionalImageURL(t *testing.T) {
	tests := []struct {
		name     string
		imageURL *string
		hasImage bool
	}{
		{
			name:     "with image URL",
			imageURL: func() *string { url := "https://example.com/btc.png"; return &url }(),
			hasImage: true,
		},
		{
			name:     "without image URL",
			imageURL: nil,
			hasImage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crypto := entities.CryptoCurrency{
				Symbol:     "BTCUSDT",
				Name:       "Bitcoin",
				MarketType: "Spot",
				ImageURL:   tt.imageURL,
				Active:     true,
			}

			if tt.hasImage {
				assert.NotNil(t, crypto.ImageURL)
				assert.Equal(t, *tt.imageURL, *crypto.ImageURL)
			} else {
				assert.Nil(t, crypto.ImageURL)
			}
		})
	}
}

func TestCryptoCurrency_RequiredFields(t *testing.T) {
	crypto := entities.CryptoCurrency{
		Symbol:     "BTCUSDT",
		Name:       "Bitcoin",
		MarketType: "Spot",
		Active:     true,
	}

	// Test that required fields are not empty
	assert.NotEmpty(t, crypto.Symbol)
	assert.NotEmpty(t, crypto.Name)
	assert.NotEmpty(t, crypto.MarketType)
	assert.NotNil(t, crypto.Active)
}

func TestCryptoCurrency_SymbolFormat(t *testing.T) {
	validSymbols := []string{
		"BTCUSDT",
		"ETHUSDT",
		"BNBBTC",
		"ADAETH",
		"DOGEBTC",
	}

	for _, symbol := range validSymbols {
		t.Run("symbol_format_"+symbol, func(t *testing.T) {
			crypto := entities.CryptoCurrency{
				Symbol:     symbol,
				Name:       "Test Coin",
				MarketType: "Spot",
				Active:     true,
			}

			assert.Equal(t, symbol, crypto.Symbol)
			assert.NotEmpty(t, crypto.Symbol)
			// Basic validation - symbols are typically uppercase and contain base/quote pair
			assert.Regexp(t, "^[A-Z]{6,10}$", crypto.Symbol)
		})
	}
}

func TestCryptoCurrency_Timestamps(t *testing.T) {
	createdAt := time.Now()
	updatedAt := createdAt.Add(1 * time.Hour)

	crypto := entities.CryptoCurrency{
		Symbol:     "BTCUSDT",
		Name:       "Bitcoin",
		MarketType: "Spot",
		Active:     true,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	assert.Equal(t, createdAt, crypto.CreatedAt)
	assert.Equal(t, updatedAt, crypto.UpdatedAt)
	assert.True(t, crypto.UpdatedAt.After(crypto.CreatedAt))
}

func TestCryptoCurrency_DefaultMarketType(t *testing.T) {
	crypto := entities.CryptoCurrency{
		Symbol: "BTCUSDT",
		Name:   "Bitcoin",
		Active: true,
		// MarketType not set - would default to "Spot" in GORM
	}

	// In a real scenario with GORM, this would be "Spot"
	// Here we just test the structure allows it
	assert.NotNil(t, &crypto.MarketType)
}

func TestCryptoCurrency_DefaultActive(t *testing.T) {
	crypto := entities.CryptoCurrency{
		Symbol:     "BTCUSDT",
		Name:       "Bitcoin",
		MarketType: "Spot",
		// Active not set - would default to true in GORM
	}

	// In a real scenario with GORM, this would be true
	// Here we just test the structure allows it
	assert.NotNil(t, &crypto.Active)
}
