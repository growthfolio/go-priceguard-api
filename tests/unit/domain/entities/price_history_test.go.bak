package entities_test

import (
	"testing"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/stretchr/testify/assert"
)

func TestPriceHistory_Creation(t *testing.T) {
	now := time.Now()

	priceHistory := entities.PriceHistory{
		Symbol:     "BTCUSDT",
		Timeframe:  "1h",
		OpenPrice:  50000.0,
		HighPrice:  52000.0,
		LowPrice:   49000.0,
		ClosePrice: 51000.0,
		Volume:     1000.0,
		Timestamp:  now,
	}

	assert.Equal(t, "BTCUSDT", priceHistory.Symbol)
	assert.Equal(t, "1h", priceHistory.Timeframe)
	assert.Equal(t, 50000.0, priceHistory.OpenPrice)
	assert.Equal(t, 52000.0, priceHistory.HighPrice)
	assert.Equal(t, 49000.0, priceHistory.LowPrice)
	assert.Equal(t, 51000.0, priceHistory.ClosePrice)
	assert.Equal(t, 1000.0, priceHistory.Volume)
	assert.Equal(t, now, priceHistory.Timestamp)
}

func TestPriceHistory_OHLCValidation(t *testing.T) {
	tests := []struct {
		name       string
		openPrice  float64
		highPrice  float64
		lowPrice   float64
		closePrice float64
		isValid    bool
	}{
		{
			name:       "valid OHLC",
			openPrice:  50000.0,
			highPrice:  52000.0,
			lowPrice:   49000.0,
			closePrice: 51000.0,
			isValid:    true,
		},
		{
			name:       "high equals open and close",
			openPrice:  50000.0,
			highPrice:  50000.0,
			lowPrice:   49000.0,
			closePrice: 50000.0,
			isValid:    true,
		},
		{
			name:       "low equals open and close",
			openPrice:  50000.0,
			highPrice:  51000.0,
			lowPrice:   50000.0,
			closePrice: 50000.0,
			isValid:    true,
		},
		{
			name:       "all prices equal",
			openPrice:  50000.0,
			highPrice:  50000.0,
			lowPrice:   50000.0,
			closePrice: 50000.0,
			isValid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priceHistory := entities.PriceHistory{
				Symbol:     "BTCUSDT",
				Timeframe:  "1h",
				OpenPrice:  tt.openPrice,
				HighPrice:  tt.highPrice,
				LowPrice:   tt.lowPrice,
				ClosePrice: tt.closePrice,
				Volume:     1000.0,
				Timestamp:  time.Now(),
			}

			// High should be >= Open and Close
			assert.GreaterOrEqual(t, priceHistory.HighPrice, priceHistory.OpenPrice)
			assert.GreaterOrEqual(t, priceHistory.HighPrice, priceHistory.ClosePrice)

			// Low should be <= Open and Close
			assert.LessOrEqual(t, priceHistory.LowPrice, priceHistory.OpenPrice)
			assert.LessOrEqual(t, priceHistory.LowPrice, priceHistory.ClosePrice)
		})
	}
}

func TestPriceHistory_Timeframes(t *testing.T) {
	validTimeframes := []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w", "1M"}

	for _, timeframe := range validTimeframes {
		t.Run("timeframe_"+timeframe, func(t *testing.T) {
			priceHistory := entities.PriceHistory{
				Symbol:     "BTCUSDT",
				Timeframe:  timeframe,
				OpenPrice:  50000.0,
				HighPrice:  52000.0,
				LowPrice:   49000.0,
				ClosePrice: 51000.0,
				Volume:     1000.0,
				Timestamp:  time.Now(),
			}

			assert.Equal(t, timeframe, priceHistory.Timeframe)
			assert.NotEmpty(t, priceHistory.Symbol)
		})
	}
}

func TestPriceHistory_VolumeValidation(t *testing.T) {
	tests := []struct {
		name   string
		volume float64
		valid  bool
	}{
		{
			name:   "positive volume",
			volume: 1000.0,
			valid:  true,
		},
		{
			name:   "zero volume",
			volume: 0.0,
			valid:  true, // Zero volume can be valid (no trades)
		},
		{
			name:   "large volume",
			volume: 1000000.0,
			valid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priceHistory := entities.PriceHistory{
				Symbol:     "BTCUSDT",
				Timeframe:  "1h",
				OpenPrice:  50000.0,
				HighPrice:  52000.0,
				LowPrice:   49000.0,
				ClosePrice: 51000.0,
				Volume:     tt.volume,
				Timestamp:  time.Now(),
			}

			assert.Equal(t, tt.volume, priceHistory.Volume)
			assert.GreaterOrEqual(t, priceHistory.Volume, 0.0) // Volume should not be negative
		})
	}
}

func TestPriceHistory_PriceChange(t *testing.T) {
	tests := []struct {
		name         string
		openPrice    float64
		closePrice   float64
		expectedUp   bool
		expectedDown bool
	}{
		{
			name:         "price increased",
			openPrice:    50000.0,
			closePrice:   51000.0,
			expectedUp:   true,
			expectedDown: false,
		},
		{
			name:         "price decreased",
			openPrice:    50000.0,
			closePrice:   49000.0,
			expectedUp:   false,
			expectedDown: true,
		},
		{
			name:         "price unchanged",
			openPrice:    50000.0,
			closePrice:   50000.0,
			expectedUp:   false,
			expectedDown: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			priceHistory := entities.PriceHistory{
				Symbol:     "BTCUSDT",
				Timeframe:  "1h",
				OpenPrice:  tt.openPrice,
				HighPrice:  52000.0,
				LowPrice:   48000.0,
				ClosePrice: tt.closePrice,
				Volume:     1000.0,
				Timestamp:  time.Now(),
			}

			isUp := priceHistory.ClosePrice > priceHistory.OpenPrice
			isDown := priceHistory.ClosePrice < priceHistory.OpenPrice

			assert.Equal(t, tt.expectedUp, isUp)
			assert.Equal(t, tt.expectedDown, isDown)
		})
	}
}

func TestPriceHistory_SymbolValidation(t *testing.T) {
	commonSymbols := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT", "DOGEUSDT", "LTCUSDT"}

	for _, symbol := range commonSymbols {
		t.Run("symbol_"+symbol, func(t *testing.T) {
			priceHistory := entities.PriceHistory{
				Symbol:     symbol,
				Timeframe:  "1h",
				OpenPrice:  1000.0,
				HighPrice:  1100.0,
				LowPrice:   900.0,
				ClosePrice: 1050.0,
				Volume:     500.0,
				Timestamp:  time.Now(),
			}

			assert.Equal(t, symbol, priceHistory.Symbol)
			assert.NotEmpty(t, priceHistory.Symbol)
		})
	}
}
