package indicators

import (
	"math"
)

// RSI calculates the Relative Strength Index
func RSI(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return nil
	}

	rsi := make([]float64, len(prices))

	// Calculate initial gains and losses
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

	// Calculate initial average gain and loss
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Calculate RSI for the first valid point
	rs := avgGain / avgLoss
	rsi[period] = 100 - (100 / (1 + rs))

	// Calculate subsequent RSI values using smoothed averages
	for i := period + 1; i < len(prices); i++ {
		avgGain = ((avgGain * float64(period-1)) + gains[i-1]) / float64(period)
		avgLoss = ((avgLoss * float64(period-1)) + losses[i-1]) / float64(period)

		if avgLoss == 0 {
			rsi[i] = 100
		} else {
			rs = avgGain / avgLoss
			rsi[i] = 100 - (100 / (1 + rs))
		}
	}

	return rsi
}

// SMA calculates Simple Moving Average
func SMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return nil
	}

	sma := make([]float64, len(prices))

	for i := period - 1; i < len(prices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j]
		}
		sma[i] = sum / float64(period)
	}

	return sma
}

// EMA calculates Exponential Moving Average
func EMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return nil
	}

	ema := make([]float64, len(prices))
	multiplier := 2.0 / float64(period+1)

	// Start with SMA for the first value
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema[period-1] = sum / float64(period)

	// Calculate EMA for subsequent values
	for i := period; i < len(prices); i++ {
		ema[i] = (prices[i] * multiplier) + (ema[i-1] * (1 - multiplier))
	}

	return ema
}

// MACD calculates Moving Average Convergence Divergence
func MACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) (macdLine, signalLine, histogram []float64) {
	if len(prices) < slowPeriod {
		return nil, nil, nil
	}

	// Calculate fast and slow EMAs
	fastEMA := EMA(prices, fastPeriod)
	slowEMA := EMA(prices, slowPeriod)

	// Calculate MACD line
	macdLine = make([]float64, len(prices))
	for i := slowPeriod - 1; i < len(prices); i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	// Calculate signal line (EMA of MACD line)
	macdValues := make([]float64, 0)
	for i := slowPeriod - 1; i < len(macdLine); i++ {
		if macdLine[i] != 0 {
			macdValues = append(macdValues, macdLine[i])
		}
	}

	if len(macdValues) >= signalPeriod {
		signalEMA := EMA(macdValues, signalPeriod)
		signalLine = make([]float64, len(prices))

		startIndex := slowPeriod - 1 + signalPeriod - 1
		for i := 0; i < len(signalEMA); i++ {
			if startIndex+i < len(signalLine) {
				signalLine[startIndex+i] = signalEMA[i]
			}
		}

		// Calculate histogram
		histogram = make([]float64, len(prices))
		for i := startIndex; i < len(prices); i++ {
			histogram[i] = macdLine[i] - signalLine[i]
		}
	}

	return macdLine, signalLine, histogram
}

// BollingerBands calculates Bollinger Bands
func BollingerBands(prices []float64, period int, stdDev float64) (upperBand, middleBand, lowerBand []float64) {
	if len(prices) < period {
		return nil, nil, nil
	}

	upperBand = make([]float64, len(prices))
	middleBand = make([]float64, len(prices))
	lowerBand = make([]float64, len(prices))

	// Calculate moving average (middle band)
	sma := SMA(prices, period)
	copy(middleBand, sma)

	// Calculate standard deviation and bands
	for i := period - 1; i < len(prices); i++ {
		// Calculate standard deviation for this period
		mean := sma[i]
		variance := 0.0

		for j := i - period + 1; j <= i; j++ {
			variance += math.Pow(prices[j]-mean, 2)
		}
		variance /= float64(period)
		stdev := math.Sqrt(variance)

		upperBand[i] = mean + (stdDev * stdev)
		lowerBand[i] = mean - (stdDev * stdev)
	}

	return upperBand, middleBand, lowerBand
}

// SuperTrend calculates SuperTrend indicator
func SuperTrend(highs, lows, closes []float64, period int, multiplier float64) (supertrend, trend []float64) {
	if len(highs) != len(lows) || len(lows) != len(closes) || len(closes) < period {
		return nil, nil
	}

	atr := ATR(highs, lows, closes, period)
	supertrend = make([]float64, len(closes))
	trend = make([]float64, len(closes))

	for i := period; i < len(closes); i++ {
		hl2 := (highs[i] + lows[i]) / 2

		upperBand := hl2 + (multiplier * atr[i])
		lowerBand := hl2 - (multiplier * atr[i])

		// Calculate trend
		if i == period {
			if closes[i] <= upperBand {
				trend[i] = 1 // Uptrend
				supertrend[i] = lowerBand
			} else {
				trend[i] = -1 // Downtrend
				supertrend[i] = upperBand
			}
		} else {
			prevSupertrend := supertrend[i-1]
			prevTrend := trend[i-1]

			if prevTrend == 1 {
				if closes[i] > lowerBand {
					trend[i] = 1
					supertrend[i] = math.Max(lowerBand, prevSupertrend)
				} else {
					trend[i] = -1
					supertrend[i] = upperBand
				}
			} else {
				if closes[i] < upperBand {
					trend[i] = -1
					supertrend[i] = math.Min(upperBand, prevSupertrend)
				} else {
					trend[i] = 1
					supertrend[i] = lowerBand
				}
			}
		}
	}

	return supertrend, trend
}

// ATR calculates Average True Range
func ATR(highs, lows, closes []float64, period int) []float64 {
	if len(highs) != len(lows) || len(lows) != len(closes) || len(closes) < period+1 {
		return nil
	}

	trueRanges := make([]float64, len(closes))

	// Calculate True Range for each period
	for i := 1; i < len(closes); i++ {
		tr1 := highs[i] - lows[i]
		tr2 := math.Abs(highs[i] - closes[i-1])
		tr3 := math.Abs(lows[i] - closes[i-1])

		trueRanges[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// Calculate ATR using RMA (Recursive Moving Average)
	atr := make([]float64, len(closes))

	// Initial ATR is SMA of true ranges
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trueRanges[i]
	}
	atr[period] = sum / float64(period)

	// Subsequent ATR values use RMA
	for i := period + 1; i < len(closes); i++ {
		atr[i] = ((atr[i-1] * float64(period-1)) + trueRanges[i]) / float64(period)
	}

	return atr
}

// Stochastic calculates Stochastic Oscillator
func Stochastic(highs, lows, closes []float64, kPeriod, dPeriod int) (percentK, percentD []float64) {
	if len(highs) != len(lows) || len(lows) != len(closes) || len(closes) < kPeriod {
		return nil, nil
	}

	percentK = make([]float64, len(closes))

	for i := kPeriod - 1; i < len(closes); i++ {
		// Find highest high and lowest low in the period
		highestHigh := highs[i-kPeriod+1]
		lowestLow := lows[i-kPeriod+1]

		for j := i - kPeriod + 2; j <= i; j++ {
			if highs[j] > highestHigh {
				highestHigh = highs[j]
			}
			if lows[j] < lowestLow {
				lowestLow = lows[j]
			}
		}

		// Calculate %K
		if highestHigh != lowestLow {
			percentK[i] = ((closes[i] - lowestLow) / (highestHigh - lowestLow)) * 100
		} else {
			percentK[i] = 50 // Neutral value when no range
		}
	}

	// Calculate %D (SMA of %K)
	percentD = SMA(percentK, dPeriod)

	return percentK, percentD
}
