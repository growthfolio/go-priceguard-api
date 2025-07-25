package services

import (
	"context"
	"fmt"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/indicators"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
	"github.com/sirupsen/logrus"
)

// TechnicalIndicatorService handles calculation and storage of technical indicators
type TechnicalIndicatorService struct {
	priceHistoryRepo       repositories.PriceHistoryRepository
	technicalIndicatorRepo repositories.TechnicalIndicatorRepository
	logger                 *logrus.Logger
}

// NewTechnicalIndicatorService creates a new technical indicator service
func NewTechnicalIndicatorService(
	priceHistoryRepo repositories.PriceHistoryRepository,
	technicalIndicatorRepo repositories.TechnicalIndicatorRepository,
	logger *logrus.Logger,
) *TechnicalIndicatorService {
	return &TechnicalIndicatorService{
		priceHistoryRepo:       priceHistoryRepo,
		technicalIndicatorRepo: technicalIndicatorRepo,
		logger:                 logger,
	}
}

// CalculateAndStoreRSI calculates RSI for a symbol and timeframe and stores it
func (s *TechnicalIndicatorService) CalculateAndStoreRSI(ctx context.Context, symbol, timeframe string, period int) error {
	// Get price history
	priceHistory, err := s.priceHistoryRepo.GetBySymbol(ctx, symbol, timeframe, period+10) // Get extra data for accuracy
	if err != nil {
		return fmt.Errorf("failed to get price history: %w", err)
	}

	if len(priceHistory) < period+1 {
		return fmt.Errorf("insufficient price data for RSI calculation")
	}

	// Extract close prices
	var closePrices []float64
	for _, ph := range priceHistory {
		closePrices = append(closePrices, ph.ClosePrice)
	}

	// Calculate RSI
	rsiResult, err := indicators.CalculateRSI(closePrices, period)
	if err != nil {
		return fmt.Errorf("failed to calculate RSI: %w", err)
	}

	// Store indicator
	indicator := &entities.TechnicalIndicator{
		Symbol:        symbol,
		Timeframe:     timeframe,
		IndicatorType: "RSI",
		Value:         &rsiResult.Value,
		Metadata: map[string]interface{}{
			"period": period,
			"signal": rsiResult.Signal,
		},
		Timestamp: time.Now(),
	}

	if err := s.technicalIndicatorRepo.Create(ctx, indicator); err != nil {
		return fmt.Errorf("failed to store RSI indicator: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":    symbol,
		"timeframe": timeframe,
		"rsi":       rsiResult.Value,
		"signal":    rsiResult.Signal,
	}).Info("RSI calculated and stored")

	return nil
}

// CalculateAndStoreEMA calculates EMA for a symbol and timeframe and stores it
func (s *TechnicalIndicatorService) CalculateAndStoreEMA(ctx context.Context, symbol, timeframe string, period int) error {
	// Get price history
	priceHistory, err := s.priceHistoryRepo.GetBySymbol(ctx, symbol, timeframe, period*2) // Get extra data for accuracy
	if err != nil {
		return fmt.Errorf("failed to get price history: %w", err)
	}

	if len(priceHistory) < period {
		return fmt.Errorf("insufficient price data for EMA calculation")
	}

	// Extract close prices
	var closePrices []float64
	for _, ph := range priceHistory {
		closePrices = append(closePrices, ph.ClosePrice)
	}

	// Calculate EMA
	emaResult, err := indicators.CalculateEMA(closePrices, period)
	if err != nil {
		return fmt.Errorf("failed to calculate EMA: %w", err)
	}

	// Store indicator
	indicator := &entities.TechnicalIndicator{
		Symbol:        symbol,
		Timeframe:     timeframe,
		IndicatorType: "EMA",
		Value:         &emaResult.Value,
		Metadata: map[string]interface{}{
			"period": period,
		},
		Timestamp: time.Now(),
	}

	if err := s.technicalIndicatorRepo.Create(ctx, indicator); err != nil {
		return fmt.Errorf("failed to store EMA indicator: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":    symbol,
		"timeframe": timeframe,
		"ema":       emaResult.Value,
		"period":    period,
	}).Info("EMA calculated and stored")

	return nil
}

// CalculateAndStoreSMA calculates SMA for a symbol and timeframe and stores it
func (s *TechnicalIndicatorService) CalculateAndStoreSMA(ctx context.Context, symbol, timeframe string, period int) error {
	// Get price history
	priceHistory, err := s.priceHistoryRepo.GetBySymbol(ctx, symbol, timeframe, period)
	if err != nil {
		return fmt.Errorf("failed to get price history: %w", err)
	}

	if len(priceHistory) < period {
		return fmt.Errorf("insufficient price data for SMA calculation")
	}

	// Extract close prices
	var closePrices []float64
	for _, ph := range priceHistory {
		closePrices = append(closePrices, ph.ClosePrice)
	}

	// Calculate SMA
	smaResult, err := indicators.CalculateSMA(closePrices, period)
	if err != nil {
		return fmt.Errorf("failed to calculate SMA: %w", err)
	}

	// Store indicator
	indicator := &entities.TechnicalIndicator{
		Symbol:        symbol,
		Timeframe:     timeframe,
		IndicatorType: "SMA",
		Value:         &smaResult.Value,
		Metadata: map[string]interface{}{
			"period": period,
		},
		Timestamp: time.Now(),
	}

	if err := s.technicalIndicatorRepo.Create(ctx, indicator); err != nil {
		return fmt.Errorf("failed to store SMA indicator: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":    symbol,
		"timeframe": timeframe,
		"sma":       smaResult.Value,
		"period":    period,
	}).Info("SMA calculated and stored")

	return nil
}

// CalculateAndStoreSuperTrend calculates SuperTrend for a symbol and timeframe and stores it
func (s *TechnicalIndicatorService) CalculateAndStoreSuperTrend(ctx context.Context, symbol, timeframe string, period int, multiplier float64) error {
	// Get price history
	priceHistory, err := s.priceHistoryRepo.GetBySymbol(ctx, symbol, timeframe, period+10)
	if err != nil {
		return fmt.Errorf("failed to get price history: %w", err)
	}

	if len(priceHistory) < period+1 {
		return fmt.Errorf("insufficient price data for SuperTrend calculation")
	}

	// Convert to price data format
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

	// Calculate SuperTrend
	stResult, err := indicators.CalculateSuperTrend(priceData, period, multiplier)
	if err != nil {
		return fmt.Errorf("failed to calculate SuperTrend: %w", err)
	}

	// Store indicator
	indicator := &entities.TechnicalIndicator{
		Symbol:        symbol,
		Timeframe:     timeframe,
		IndicatorType: "SuperTrend",
		Value:         &stResult.Value,
		Metadata: map[string]interface{}{
			"period":     period,
			"multiplier": multiplier,
			"trend":      stResult.Trend,
			"upper_band": stResult.UpperBand,
			"lower_band": stResult.LowerBand,
			"atr":        stResult.ATR,
		},
		Timestamp: time.Now(),
	}

	if err := s.technicalIndicatorRepo.Create(ctx, indicator); err != nil {
		return fmt.Errorf("failed to store SuperTrend indicator: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":     symbol,
		"timeframe":  timeframe,
		"supertrend": stResult.Value,
		"trend":      stResult.Trend,
	}).Info("SuperTrend calculated and stored")

	return nil
}

// CalculateAndStoreBollingerBands calculates Bollinger Bands for a symbol and timeframe and stores it
func (s *TechnicalIndicatorService) CalculateAndStoreBollingerBands(ctx context.Context, symbol, timeframe string, period int, multiplier float64) error {
	// Get price history
	priceHistory, err := s.priceHistoryRepo.GetBySymbol(ctx, symbol, timeframe, period)
	if err != nil {
		return fmt.Errorf("failed to get price history: %w", err)
	}

	if len(priceHistory) < period {
		return fmt.Errorf("insufficient price data for Bollinger Bands calculation")
	}

	// Extract close prices
	var closePrices []float64
	for _, ph := range priceHistory {
		closePrices = append(closePrices, ph.ClosePrice)
	}

	// Calculate Bollinger Bands
	bbResult, err := indicators.CalculateBollingerBands(closePrices, period, multiplier)
	if err != nil {
		return fmt.Errorf("failed to calculate Bollinger Bands: %w", err)
	}

	// Store upper band
	upperValue := bbResult["upper_band"]
	upperIndicator := &entities.TechnicalIndicator{
		Symbol:        symbol,
		Timeframe:     timeframe,
		IndicatorType: "BB_Upper",
		Value:         &upperValue,
		Metadata: map[string]interface{}{
			"period":     period,
			"multiplier": multiplier,
			"sma":        bbResult["sma"],
			"std_dev":    bbResult["std_dev"],
		},
		Timestamp: time.Now(),
	}

	// Store middle band (SMA)
	middleValue := bbResult["sma"]
	middleIndicator := &entities.TechnicalIndicator{
		Symbol:        symbol,
		Timeframe:     timeframe,
		IndicatorType: "BB_Middle",
		Value:         &middleValue,
		Metadata: map[string]interface{}{
			"period":     period,
			"multiplier": multiplier,
		},
		Timestamp: time.Now(),
	}

	// Store lower band
	lowerValue := bbResult["lower_band"]
	lowerIndicator := &entities.TechnicalIndicator{
		Symbol:        symbol,
		Timeframe:     timeframe,
		IndicatorType: "BB_Lower",
		Value:         &lowerValue,
		Metadata: map[string]interface{}{
			"period":     period,
			"multiplier": multiplier,
			"sma":        bbResult["sma"],
			"std_dev":    bbResult["std_dev"],
		},
		Timestamp: time.Now(),
	}

	// Store all indicators
	if err := s.technicalIndicatorRepo.Create(ctx, upperIndicator); err != nil {
		return fmt.Errorf("failed to store BB upper indicator: %w", err)
	}
	if err := s.technicalIndicatorRepo.Create(ctx, middleIndicator); err != nil {
		return fmt.Errorf("failed to store BB middle indicator: %w", err)
	}
	if err := s.technicalIndicatorRepo.Create(ctx, lowerIndicator); err != nil {
		return fmt.Errorf("failed to store BB lower indicator: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":     symbol,
		"timeframe":  timeframe,
		"upper_band": bbResult["upper_band"],
		"middle":     bbResult["sma"],
		"lower_band": bbResult["lower_band"],
	}).Info("Bollinger Bands calculated and stored")

	return nil
}

// CalculateAllIndicators calculates all indicators for a symbol and timeframe
func (s *TechnicalIndicatorService) CalculateAllIndicators(ctx context.Context, symbol, timeframe string) error {
	s.logger.WithFields(logrus.Fields{
		"symbol":    symbol,
		"timeframe": timeframe,
	}).Info("Calculating all indicators")

	// Calculate RSI (14 period)
	if err := s.CalculateAndStoreRSI(ctx, symbol, timeframe, 14); err != nil {
		s.logger.WithError(err).Error("Failed to calculate RSI")
	}

	// Calculate EMAs (12, 26 period)
	if err := s.CalculateAndStoreEMA(ctx, symbol, timeframe, 12); err != nil {
		s.logger.WithError(err).Error("Failed to calculate EMA 12")
	}
	if err := s.CalculateAndStoreEMA(ctx, symbol, timeframe, 26); err != nil {
		s.logger.WithError(err).Error("Failed to calculate EMA 26")
	}

	// Calculate SMAs (20, 50 period)
	if err := s.CalculateAndStoreSMA(ctx, symbol, timeframe, 20); err != nil {
		s.logger.WithError(err).Error("Failed to calculate SMA 20")
	}
	if err := s.CalculateAndStoreSMA(ctx, symbol, timeframe, 50); err != nil {
		s.logger.WithError(err).Error("Failed to calculate SMA 50")
	}

	// Calculate SuperTrend (10 period, 3.0 multiplier)
	if err := s.CalculateAndStoreSuperTrend(ctx, symbol, timeframe, 10, 3.0); err != nil {
		s.logger.WithError(err).Error("Failed to calculate SuperTrend")
	}

	// Calculate Bollinger Bands (20 period, 2.0 multiplier)
	if err := s.CalculateAndStoreBollingerBands(ctx, symbol, timeframe, 20, 2.0); err != nil {
		s.logger.WithError(err).Error("Failed to calculate Bollinger Bands")
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":    symbol,
		"timeframe": timeframe,
	}).Info("All indicators calculated")

	return nil
}

// GetLatestIndicators gets the latest calculated indicators for a symbol and timeframe
func (s *TechnicalIndicatorService) GetLatestIndicators(ctx context.Context, symbol, timeframe string) (map[string]*entities.TechnicalIndicator, error) {
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
