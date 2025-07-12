package indicators

import (
	"fmt"
	"math"
)

// PriceData represents price data for technical indicators
type PriceData struct {
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// RSIResult represents RSI calculation result
type RSIResult struct {
	Value     float64
	Signal    string // "oversold", "overbought", "neutral"
	Timestamp int64
}

// EMAResult represents EMA calculation result
type EMAResult struct {
	Value     float64
	Period    int
	Timestamp int64
}

// SMAResult represents SMA calculation result
type SMAResult struct {
	Value     float64
	Period    int
	Timestamp int64
}

// SuperTrendResult represents SuperTrend calculation result
type SuperTrendResult struct {
	Value      float64
	Trend      string // "up", "down"
	UpperBand  float64
	LowerBand  float64
	ATR        float64
	Multiplier float64
	Timestamp  int64
}

// TrueRangeResult represents True Range calculation result
type TrueRangeResult struct {
	Value     float64
	Timestamp int64
}

// CalculateRSI calculates the Relative Strength Index
func CalculateRSI(prices []float64, period int) (*RSIResult, error) {
	if len(prices) < period+1 {
		return nil, fmt.Errorf("insufficient data: need at least %d prices, got %d", period+1, len(prices))
	}

	// Calculate price changes
	var gains, losses []float64
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	if len(gains) < period {
		return nil, fmt.Errorf("insufficient data for RSI calculation")
	}

	// Calculate initial average gain and loss
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Calculate smoothed averages
	for i := period; i < len(gains); i++ {
		avgGain = ((avgGain * float64(period-1)) + gains[i]) / float64(period)
		avgLoss = ((avgLoss * float64(period-1)) + losses[i]) / float64(period)
	}

	// Calculate RSI
	if avgLoss == 0 {
		return &RSIResult{
			Value:  100,
			Signal: "overbought",
		}, nil
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	// Determine signal
	var signal string
	if rsi > 70 {
		signal = "overbought"
	} else if rsi < 30 {
		signal = "oversold"
	} else {
		signal = "neutral"
	}

	return &RSIResult{
		Value:  rsi,
		Signal: signal,
	}, nil
}

// CalculateEMA calculates the Exponential Moving Average
func CalculateEMA(prices []float64, period int) (*EMAResult, error) {
	if len(prices) < period {
		return nil, fmt.Errorf("insufficient data: need at least %d prices, got %d", period, len(prices))
	}

	// Calculate multiplier
	multiplier := 2.0 / (float64(period) + 1.0)

	// Start with SMA for the first EMA value
	var sma float64
	for i := 0; i < period; i++ {
		sma += prices[i]
	}
	sma /= float64(period)

	ema := sma

	// Calculate EMA for remaining periods
	for i := period; i < len(prices); i++ {
		ema = (prices[i] * multiplier) + (ema * (1 - multiplier))
	}

	return &EMAResult{
		Value:  ema,
		Period: period,
	}, nil
}

// CalculateSMA calculates the Simple Moving Average
func CalculateSMA(prices []float64, period int) (*SMAResult, error) {
	if len(prices) < period {
		return nil, fmt.Errorf("insufficient data: need at least %d prices, got %d", period, len(prices))
	}

	var sum float64
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}

	sma := sum / float64(period)

	return &SMAResult{
		Value:  sma,
		Period: period,
	}, nil
}

// CalculateTrueRange calculates the True Range
func CalculateTrueRange(current, previous PriceData) *TrueRangeResult {
	tr1 := current.High - current.Low
	tr2 := math.Abs(current.High - previous.Close)
	tr3 := math.Abs(current.Low - previous.Close)

	tr := math.Max(tr1, math.Max(tr2, tr3))

	return &TrueRangeResult{
		Value: tr,
	}
}

// CalculateATR calculates the Average True Range
func CalculateATR(priceData []PriceData, period int) (float64, error) {
	if len(priceData) < period+1 {
		return 0, fmt.Errorf("insufficient data: need at least %d price data points, got %d", period+1, len(priceData))
	}

	var trValues []float64
	for i := 1; i < len(priceData); i++ {
		tr := CalculateTrueRange(priceData[i], priceData[i-1])
		trValues = append(trValues, tr.Value)
	}

	// Calculate initial ATR as SMA of True Range
	var sum float64
	for i := 0; i < period; i++ {
		sum += trValues[i]
	}
	atr := sum / float64(period)

	// Smooth ATR using Wilder's smoothing (similar to EMA with α = 1/period)
	for i := period; i < len(trValues); i++ {
		atr = ((atr * float64(period-1)) + trValues[i]) / float64(period)
	}

	return atr, nil
}

// CalculateSuperTrend calculates the SuperTrend indicator
func CalculateSuperTrend(priceData []PriceData, period int, multiplier float64) (*SuperTrendResult, error) {
	if len(priceData) < period+1 {
		return nil, fmt.Errorf("insufficient data: need at least %d price data points, got %d", period+1, len(priceData))
	}

	// Calculate ATR
	atr, err := CalculateATR(priceData, period)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate ATR: %w", err)
	}

	// Get the latest price data
	latest := priceData[len(priceData)-1]
	hl2 := (latest.High + latest.Low) / 2

	// Calculate basic upper and lower bands
	basicUpperBand := hl2 + (multiplier * atr)
	basicLowerBand := hl2 - (multiplier * atr)

	// For simplicity, we'll use the basic bands directly
	// In a full implementation, you'd need to track previous values for proper SuperTrend calculation
	upperBand := basicUpperBand
	lowerBand := basicLowerBand

	// Determine trend
	var trend string
	var superTrendValue float64

	if latest.Close > lowerBand {
		trend = "up"
		superTrendValue = lowerBand
	} else if latest.Close < upperBand {
		trend = "down"
		superTrendValue = upperBand
	} else {
		// Use previous trend or default to up
		trend = "up"
		superTrendValue = lowerBand
	}

	return &SuperTrendResult{
		Value:      superTrendValue,
		Trend:      trend,
		UpperBand:  upperBand,
		LowerBand:  lowerBand,
		ATR:        atr,
		Multiplier: multiplier,
	}, nil
}

// CalculateStochastic calculates the Stochastic Oscillator
func CalculateStochastic(priceData []PriceData, kPeriod, dPeriod int) (map[string]float64, error) {
	if len(priceData) < kPeriod {
		return nil, fmt.Errorf("insufficient data: need at least %d price data points, got %d", kPeriod, len(priceData))
	}

	// Get the last kPeriod prices
	recentPrices := priceData[len(priceData)-kPeriod:]

	// Find highest high and lowest low in the period
	var highest, lowest float64
	highest = recentPrices[0].High
	lowest = recentPrices[0].Low

	for _, price := range recentPrices {
		if price.High > highest {
			highest = price.High
		}
		if price.Low < lowest {
			lowest = price.Low
		}
	}

	// Calculate %K
	currentClose := priceData[len(priceData)-1].Close
	kPercent := ((currentClose - lowest) / (highest - lowest)) * 100

	// For %D, we'd need multiple %K values to calculate the moving average
	// For simplicity, we'll return just %K
	return map[string]float64{
		"k_percent": kPercent,
		"highest":   highest,
		"lowest":    lowest,
	}, nil
}

// CalculateMACD calculates the MACD (Moving Average Convergence Divergence)
func CalculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) (map[string]float64, error) {
	if len(prices) < slowPeriod {
		return nil, fmt.Errorf("insufficient data: need at least %d prices, got %d", slowPeriod, len(prices))
	}

	// Calculate fast EMA
	fastEMA, err := CalculateEMA(prices, fastPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fast EMA: %w", err)
	}

	// Calculate slow EMA
	slowEMA, err := CalculateEMA(prices, slowPeriod)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate slow EMA: %w", err)
	}

	// Calculate MACD line
	macdLine := fastEMA.Value - slowEMA.Value

	// For signal line, we'd need multiple MACD values to calculate EMA
	// For simplicity, we'll return just the MACD line
	return map[string]float64{
		"macd":     macdLine,
		"fast_ema": fastEMA.Value,
		"slow_ema": slowEMA.Value,
	}, nil
}

// CalculateBollingerBands calculates Bollinger Bands
func CalculateBollingerBands(prices []float64, period int, multiplier float64) (map[string]float64, error) {
	if len(prices) < period {
		return nil, fmt.Errorf("insufficient data: need at least %d prices, got %d", period, len(prices))
	}

	// Calculate SMA
	sma, err := CalculateSMA(prices, period)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate SMA: %w", err)
	}

	// Calculate standard deviation
	recentPrices := prices[len(prices)-period:]
	var variance float64
	for _, price := range recentPrices {
		variance += math.Pow(price-sma.Value, 2)
	}
	variance /= float64(period)
	stdDev := math.Sqrt(variance)

	// Calculate bands
	upperBand := sma.Value + (multiplier * stdDev)
	lowerBand := sma.Value - (multiplier * stdDev)

	return map[string]float64{
		"sma":        sma.Value,
		"upper_band": upperBand,
		"lower_band": lowerBand,
		"std_dev":    stdDev,
	}, nil
}

// ValidateTimeframe checks if the timeframe is valid
func ValidateTimeframe(timeframe string) bool {
	validTimeframes := map[string]bool{
		"1m":  true,
		"5m":  true,
		"15m": true,
		"30m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
		"1w":  true,
	}
	return validTimeframes[timeframe]
}

// GetTimeframeMilliseconds returns the timeframe duration in milliseconds
func GetTimeframeMilliseconds(timeframe string) int64 {
	timeframes := map[string]int64{
		"1m":  60 * 1000,
		"5m":  5 * 60 * 1000,
		"15m": 15 * 60 * 1000,
		"30m": 30 * 60 * 1000,
		"1h":  60 * 60 * 1000,
		"4h":  4 * 60 * 60 * 1000,
		"1d":  24 * 60 * 60 * 1000,
		"1w":  7 * 24 * 60 * 60 * 1000,
	}
	return timeframes[timeframe]
}
