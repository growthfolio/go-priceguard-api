package entities_test

import (
	"testing"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/stretchr/testify/assert"
)

func TestTechnicalIndicator_Creation(t *testing.T) {
	now := time.Now()
	value := 65.5

	indicator := entities.TechnicalIndicator{
		Symbol:        "BTCUSDT",
		Timeframe:     "1h",
		IndicatorType: "rsi",
		Value:         &value,
		Timestamp:     now,
	}

	assert.Equal(t, "BTCUSDT", indicator.Symbol)
	assert.Equal(t, "1h", indicator.Timeframe)
	assert.Equal(t, "rsi", indicator.IndicatorType)
	assert.NotNil(t, indicator.Value)
	assert.Equal(t, 65.5, *indicator.Value)
	assert.Equal(t, now, indicator.Timestamp)
}

func TestTechnicalIndicator_Types(t *testing.T) {
	indicatorTypes := []struct {
		name  string
		value float64
	}{
		{"rsi", 70.5},          // Relative Strength Index (0-100)
		{"ema", 50000.0},       // Exponential Moving Average
		{"sma", 49500.0},       // Simple Moving Average
		{"macd", 150.25},       // MACD
		{"bollinger", 51000.0}, // Bollinger Bands
		{"stoch", 85.2},        // Stochastic Oscillator
		{"atr", 2500.0},        // Average True Range
		{"adx", 45.8},          // Average Directional Index
	}

	for _, indicatorType := range indicatorTypes {
		t.Run("indicator_"+indicatorType.name, func(t *testing.T) {
			indicator := entities.TechnicalIndicator{
				Symbol:        "BTCUSDT",
				Timeframe:     "1h",
				IndicatorType: indicatorType.name,
				Value:         &indicatorType.value,
				Timestamp:     time.Now(),
			}

			assert.Equal(t, indicatorType.name, indicator.IndicatorType)
			assert.NotNil(t, indicator.Value)
			assert.Equal(t, indicatorType.value, *indicator.Value)
		})
	}
}

func TestTechnicalIndicator_Timeframes(t *testing.T) {
	value := 70.0
	timeframes := []string{"1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w"}

	for _, timeframe := range timeframes {
		t.Run("timeframe_"+timeframe, func(t *testing.T) {
			indicator := entities.TechnicalIndicator{
				Symbol:        "BTCUSDT",
				Timeframe:     timeframe,
				IndicatorType: "rsi",
				Value:         &value,
				Timestamp:     time.Now(),
			}

			assert.Equal(t, timeframe, indicator.Timeframe)
		})
	}
}

func TestTechnicalIndicator_RSIValues(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		isValid bool
		signal  string
	}{
		{
			name:    "oversold RSI",
			value:   25.0,
			isValid: true,
			signal:  "oversold",
		},
		{
			name:    "neutral RSI",
			value:   50.0,
			isValid: true,
			signal:  "neutral",
		},
		{
			name:    "overbought RSI",
			value:   75.0,
			isValid: true,
			signal:  "overbought",
		},
		{
			name:    "minimum RSI",
			value:   0.0,
			isValid: true,
			signal:  "extreme_oversold",
		},
		{
			name:    "maximum RSI",
			value:   100.0,
			isValid: true,
			signal:  "extreme_overbought",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indicator := entities.TechnicalIndicator{
				Symbol:        "BTCUSDT",
				Timeframe:     "1h",
				IndicatorType: "rsi",
				Value:         &tt.value,
				Timestamp:     time.Now(),
			}

			assert.Equal(t, "rsi", indicator.IndicatorType)
			assert.NotNil(t, indicator.Value)
			assert.Equal(t, tt.value, *indicator.Value)

			// RSI should be between 0 and 100
			assert.GreaterOrEqual(t, *indicator.Value, 0.0)
			assert.LessOrEqual(t, *indicator.Value, 100.0)
		})
	}
}

func TestTechnicalIndicator_NullValue(t *testing.T) {
	indicator := entities.TechnicalIndicator{
		Symbol:        "BTCUSDT",
		Timeframe:     "1h",
		IndicatorType: "rsi",
		Value:         nil, // No value calculated yet
		Timestamp:     time.Now(),
	}

	assert.Equal(t, "rsi", indicator.IndicatorType)
	assert.Nil(t, indicator.Value)
	assert.NotZero(t, indicator.Timestamp)
}

func TestTechnicalIndicator_MACD(t *testing.T) {
	tests := []struct {
		name   string
		value  float64
		signal string
	}{
		{
			name:   "positive MACD",
			value:  150.25,
			signal: "bullish",
		},
		{
			name:   "negative MACD",
			value:  -75.50,
			signal: "bearish",
		},
		{
			name:   "zero MACD",
			value:  0.0,
			signal: "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indicator := entities.TechnicalIndicator{
				Symbol:        "BTCUSDT",
				Timeframe:     "1h",
				IndicatorType: "macd",
				Value:         &tt.value,
				Timestamp:     time.Now(),
			}

			assert.Equal(t, "macd", indicator.IndicatorType)
			assert.NotNil(t, indicator.Value)
			assert.Equal(t, tt.value, *indicator.Value)
		})
	}
}

func TestTechnicalIndicator_MovingAverages(t *testing.T) {
	tests := []struct {
		name          string
		indicatorType string
		price         float64
		value         float64
		signal        string
	}{
		{
			name:          "EMA above price",
			indicatorType: "ema",
			price:         50000.0,
			value:         51000.0,
			signal:        "resistance",
		},
		{
			name:          "EMA below price",
			indicatorType: "ema",
			price:         50000.0,
			value:         49000.0,
			signal:        "support",
		},
		{
			name:          "SMA equals price",
			indicatorType: "sma",
			price:         50000.0,
			value:         50000.0,
			signal:        "neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indicator := entities.TechnicalIndicator{
				Symbol:        "BTCUSDT",
				Timeframe:     "1h",
				IndicatorType: tt.indicatorType,
				Value:         &tt.value,
				Timestamp:     time.Now(),
			}

			assert.Equal(t, tt.indicatorType, indicator.IndicatorType)
			assert.NotNil(t, indicator.Value)
			assert.Equal(t, tt.value, *indicator.Value)
			assert.Positive(t, *indicator.Value) // Prices/MAs should be positive
		})
	}
}

func TestTechnicalIndicator_MultipleSymbols(t *testing.T) {
	symbols := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT", "DOGEUSDT"}
	value := 70.0

	for _, symbol := range symbols {
		t.Run("symbol_"+symbol, func(t *testing.T) {
			indicator := entities.TechnicalIndicator{
				Symbol:        symbol,
				Timeframe:     "1h",
				IndicatorType: "rsi",
				Value:         &value,
				Timestamp:     time.Now(),
			}

			assert.Equal(t, symbol, indicator.Symbol)
			assert.NotEmpty(t, indicator.Symbol)
		})
	}
}

func TestTechnicalIndicator_Timestamp(t *testing.T) {
	now := time.Now()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(1 * time.Hour)
	value := 70.0

	tests := []struct {
		name      string
		timestamp time.Time
	}{
		{
			name:      "current timestamp",
			timestamp: now,
		},
		{
			name:      "past timestamp",
			timestamp: pastTime,
		},
		{
			name:      "future timestamp",
			timestamp: futureTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indicator := entities.TechnicalIndicator{
				Symbol:        "BTCUSDT",
				Timeframe:     "1h",
				IndicatorType: "rsi",
				Value:         &value,
				Timestamp:     tt.timestamp,
			}

			assert.Equal(t, tt.timestamp, indicator.Timestamp)
			assert.NotZero(t, indicator.Timestamp)
		})
	}
}

func TestTechnicalIndicator_RequiredFields(t *testing.T) {
	value := 70.0
	indicator := entities.TechnicalIndicator{
		Symbol:        "BTCUSDT",
		Timeframe:     "1h",
		IndicatorType: "rsi",
		Value:         &value,
		Timestamp:     time.Now(),
	}

	// Test that required fields are not empty
	assert.NotEmpty(t, indicator.Symbol)
	assert.NotEmpty(t, indicator.Timeframe)
	assert.NotEmpty(t, indicator.IndicatorType)
	assert.NotZero(t, indicator.Timestamp)
	// Value can be nil in some cases, so we don't require it
}
