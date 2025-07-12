package services

import (
	"context"
	"fmt"
	"time"

	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/indicators"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/repositories"
	"github.com/sirupsen/logrus"
)

// PullbackEntryService handles pullback entry signal detection
type PullbackEntryService struct {
	priceHistoryRepo       repositories.PriceHistoryRepository
	technicalIndicatorRepo repositories.TechnicalIndicatorRepository
	logger                 *logrus.Logger
}

// PullbackEntry represents a pullback entry signal
type PullbackEntry struct {
	Symbol      string    `json:"symbol"`
	Timeframe   string    `json:"timeframe"`
	Signal      string    `json:"signal"`     // "LONG", "SHORT", "NEUTRAL"
	Confidence  float64   `json:"confidence"` // 0-100
	EntryPrice  float64   `json:"entry_price"`
	StopLoss    float64   `json:"stop_loss"`
	TakeProfit1 float64   `json:"take_profit_1"`
	TakeProfit2 float64   `json:"take_profit_2"`
	RSI         float64   `json:"rsi"`
	EMATrend    string    `json:"ema_trend"`
	SuperTrend  string    `json:"supertrend"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewPullbackEntryService creates a new pullback entry service
func NewPullbackEntryService(
	priceHistoryRepo repositories.PriceHistoryRepository,
	technicalIndicatorRepo repositories.TechnicalIndicatorRepository,
	logger *logrus.Logger,
) *PullbackEntryService {
	return &PullbackEntryService{
		priceHistoryRepo:       priceHistoryRepo,
		technicalIndicatorRepo: technicalIndicatorRepo,
		logger:                 logger,
	}
}

// AnalyzePullbackEntry analyzes and generates pullback entry signals
func (s *PullbackEntryService) AnalyzePullbackEntry(ctx context.Context, symbol, timeframe string) (*PullbackEntry, error) {
	s.logger.WithFields(logrus.Fields{
		"symbol":    symbol,
		"timeframe": timeframe,
	}).Info("Analyzing pullback entry")

	// Get recent price history
	priceHistory, err := s.priceHistoryRepo.GetBySymbol(ctx, symbol, timeframe, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get price history: %w", err)
	}

	if len(priceHistory) < 20 {
		return nil, fmt.Errorf("insufficient price data for pullback analysis")
	}

	// Get current price
	currentPrice := priceHistory[len(priceHistory)-1].ClosePrice

	// Get latest technical indicators
	indicators, err := s.getLatestIndicators(ctx, symbol, timeframe)
	if err != nil {
		return nil, fmt.Errorf("failed to get indicators: %w", err)
	}

	// Analyze pullback entry
	signal := s.analyzePullbackSignal(priceHistory, indicators)

	// Calculate entry levels
	entry := &PullbackEntry{
		Symbol:     symbol,
		Timeframe:  timeframe,
		Signal:     signal.Signal,
		Confidence: signal.Confidence,
		EntryPrice: currentPrice,
		RSI:        signal.RSI,
		EMATrend:   signal.EMATrend,
		SuperTrend: signal.SuperTrend,
		Timestamp:  time.Now(),
	}

	// Calculate risk management levels
	s.calculateRiskLevels(entry, priceHistory)

	return entry, nil
}

// PullbackSignal represents the analysis result
type PullbackSignal struct {
	Signal     string
	Confidence float64
	RSI        float64
	EMATrend   string
	SuperTrend string
}

// analyzePullbackSignal analyzes price action and indicators for pullback signals
func (s *PullbackEntryService) analyzePullbackSignal(priceHistory []entities.PriceHistory, indicators map[string]*entities.TechnicalIndicator) *PullbackSignal {
	signal := &PullbackSignal{
		Signal:     "NEUTRAL",
		Confidence: 0,
	}

	// Analyze RSI
	if rsiIndicator, exists := indicators["RSI"]; exists && rsiIndicator.Value != nil {
		signal.RSI = *rsiIndicator.Value

		// RSI oversold condition for long entries
		if signal.RSI < 30 {
			signal.Signal = "LONG"
			signal.Confidence += 25
		}
		// RSI overbought condition for short entries
		if signal.RSI > 70 {
			signal.Signal = "SHORT"
			signal.Confidence += 25
		}
		// RSI in neutral zone
		if signal.RSI >= 40 && signal.RSI <= 60 {
			signal.Confidence += 10
		}
	}

	// Analyze EMA trend
	ema12 := s.getIndicatorValue(indicators, "EMA", 12)
	ema26 := s.getIndicatorValue(indicators, "EMA", 26)

	if ema12 != nil && ema26 != nil {
		if *ema12 > *ema26 {
			signal.EMATrend = "BULLISH"
			if signal.Signal == "LONG" || signal.Signal == "NEUTRAL" {
				signal.Confidence += 20
			}
		} else {
			signal.EMATrend = "BEARISH"
			if signal.Signal == "SHORT" || signal.Signal == "NEUTRAL" {
				signal.Confidence += 20
			}
		}
	}

	// Analyze SuperTrend
	if stIndicator, exists := indicators["SuperTrend"]; exists && stIndicator.Metadata != nil {
		if trend, ok := stIndicator.Metadata["trend"].(string); ok {
			signal.SuperTrend = trend

			if trend == "up" && signal.Signal == "LONG" {
				signal.Confidence += 25
			}
			if trend == "down" && signal.Signal == "SHORT" {
				signal.Confidence += 25
			}
		}
	}

	// Analyze price action patterns
	signal.Confidence += s.analyzePriceAction(priceHistory)

	// Ensure confidence doesn't exceed 100
	if signal.Confidence > 100 {
		signal.Confidence = 100
	}

	// Minimum confidence threshold
	if signal.Confidence < 30 {
		signal.Signal = "NEUTRAL"
	}

	return signal
}

// analyzePriceAction analyzes price action patterns for additional confirmation
func (s *PullbackEntryService) analyzePriceAction(priceHistory []entities.PriceHistory) float64 {
	if len(priceHistory) < 10 {
		return 0
	}

	confidence := 0.0
	recent := priceHistory[len(priceHistory)-5:]

	// Check for higher lows (bullish pattern)
	higherLows := true
	for i := 1; i < len(recent); i++ {
		if recent[i].LowPrice < recent[i-1].LowPrice {
			higherLows = false
			break
		}
	}
	if higherLows {
		confidence += 15
	}

	// Check for lower highs (bearish pattern)
	lowerHighs := true
	for i := 1; i < len(recent); i++ {
		if recent[i].HighPrice > recent[i-1].HighPrice {
			lowerHighs = false
			break
		}
	}
	if lowerHighs {
		confidence += 15
	}

	// Check for volume confirmation
	avgVolume := 0.0
	for _, ph := range recent {
		avgVolume += ph.Volume
	}
	avgVolume /= float64(len(recent))

	lastVolume := recent[len(recent)-1].Volume
	if lastVolume > avgVolume*1.2 {
		confidence += 10
	}

	return confidence
}

// getIndicatorValue gets indicator value by type and period
func (s *PullbackEntryService) getIndicatorValue(indicators map[string]*entities.TechnicalIndicator, indicatorType string, period int) *float64 {
	for _, indicator := range indicators {
		if indicator.IndicatorType == indicatorType {
			if metadata, ok := indicator.Metadata["period"].(float64); ok && int(metadata) == period {
				return indicator.Value
			}
		}
	}
	return nil
}

// calculateRiskLevels calculates stop loss and take profit levels
func (s *PullbackEntryService) calculateRiskLevels(entry *PullbackEntry, priceHistory []entities.PriceHistory) {
	if len(priceHistory) < 10 {
		return
	}

	// Calculate ATR for volatility-based levels
	atr := s.calculateATR(priceHistory, 14)

	if entry.Signal == "LONG" {
		// Stop loss below recent low with ATR buffer
		recentLow := s.findRecentLow(priceHistory, 10)
		entry.StopLoss = recentLow - (atr * 0.5)

		// Take profit levels
		entry.TakeProfit1 = entry.EntryPrice + (atr * 1.5)
		entry.TakeProfit2 = entry.EntryPrice + (atr * 3.0)

	} else if entry.Signal == "SHORT" {
		// Stop loss above recent high with ATR buffer
		recentHigh := s.findRecentHigh(priceHistory, 10)
		entry.StopLoss = recentHigh + (atr * 0.5)

		// Take profit levels
		entry.TakeProfit1 = entry.EntryPrice - (atr * 1.5)
		entry.TakeProfit2 = entry.EntryPrice - (atr * 3.0)
	}
}

// calculateATR calculates Average True Range
func (s *PullbackEntryService) calculateATR(priceHistory []entities.PriceHistory, period int) float64 {
	if len(priceHistory) < period+1 {
		return 0
	}

	var priceData []indicators.PriceData
	for _, ph := range priceHistory {
		priceData = append(priceData, indicators.PriceData{
			Open:   ph.OpenPrice,
			High:   ph.HighPrice,
			Low:    ph.LowPrice,
			Close:  ph.ClosePrice,
			Volume: ph.Volume,
		})
	}

	atr, err := indicators.CalculateATR(priceData, period)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to calculate ATR")
		return 0
	}

	return atr
}

// findRecentLow finds the lowest low in recent periods
func (s *PullbackEntryService) findRecentLow(priceHistory []entities.PriceHistory, periods int) float64 {
	if len(priceHistory) == 0 {
		return 0
	}

	start := len(priceHistory) - periods
	if start < 0 {
		start = 0
	}

	lowest := priceHistory[start].LowPrice
	for i := start; i < len(priceHistory); i++ {
		if priceHistory[i].LowPrice < lowest {
			lowest = priceHistory[i].LowPrice
		}
	}

	return lowest
}

// findRecentHigh finds the highest high in recent periods
func (s *PullbackEntryService) findRecentHigh(priceHistory []entities.PriceHistory, periods int) float64 {
	if len(priceHistory) == 0 {
		return 0
	}

	start := len(priceHistory) - periods
	if start < 0 {
		start = 0
	}

	highest := priceHistory[start].HighPrice
	for i := start; i < len(priceHistory); i++ {
		if priceHistory[i].HighPrice > highest {
			highest = priceHistory[i].HighPrice
		}
	}

	return highest
}

// getLatestIndicators retrieves latest technical indicators
func (s *PullbackEntryService) getLatestIndicators(ctx context.Context, symbol, timeframe string) (map[string]*entities.TechnicalIndicator, error) {
	indicators := map[string]*entities.TechnicalIndicator{}

	indicatorTypes := []string{"RSI", "EMA", "SMA", "SuperTrend", "BB_Upper", "BB_Middle", "BB_Lower"}

	for _, indicatorType := range indicatorTypes {
		indicator, err := s.technicalIndicatorRepo.GetLatest(ctx, symbol, timeframe, indicatorType)
		if err != nil {
			s.logger.WithError(err).WithField("indicator_type", indicatorType).Warn("Failed to get latest indicator")
			continue
		}
		indicators[indicatorType] = indicator
	}

	return indicators, nil
}

// GetPullbackEntriesForTimeframes gets pullback entries for multiple timeframes
func (s *PullbackEntryService) GetPullbackEntriesForTimeframes(ctx context.Context, symbol string, timeframes []string) (map[string]*PullbackEntry, error) {
	entries := make(map[string]*PullbackEntry)

	for _, timeframe := range timeframes {
		entry, err := s.AnalyzePullbackEntry(ctx, symbol, timeframe)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"symbol":    symbol,
				"timeframe": timeframe,
			}).Warn("Failed to analyze pullback entry")
			continue
		}
		entries[timeframe] = entry
	}

	return entries, nil
}
