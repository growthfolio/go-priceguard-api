package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/growthfolio/go-priceguard-api/internal/domain/repositories"
	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/external"
	"github.com/sirupsen/logrus"
)

// CryptoDataService handles cryptocurrency data collection and management
type CryptoDataService struct {
	binanceClient          *external.BinanceClient
	cryptoRepo             repositories.CryptoCurrencyRepository
	priceHistoryRepo       repositories.PriceHistoryRepository
	technicalIndicatorRepo repositories.TechnicalIndicatorRepository
	logger                 *logrus.Logger

	// Internal state
	mu             sync.RWMutex
	isCollecting   bool
	stopChan       chan struct{}
	updateInterval time.Duration
}

// NewCryptoDataService creates a new crypto data service
func NewCryptoDataService(
	binanceClient *external.BinanceClient,
	cryptoRepo repositories.CryptoCurrencyRepository,
	priceHistoryRepo repositories.PriceHistoryRepository,
	technicalIndicatorRepo repositories.TechnicalIndicatorRepository,
	logger *logrus.Logger,
) *CryptoDataService {
	return &CryptoDataService{
		binanceClient:          binanceClient,
		cryptoRepo:             cryptoRepo,
		priceHistoryRepo:       priceHistoryRepo,
		technicalIndicatorRepo: technicalIndicatorRepo,
		logger:                 logger,
		updateInterval:         30 * time.Second, // Default 30 seconds
		stopChan:               make(chan struct{}),
	}
}

// StartDataCollection starts the background data collection process
func (s *CryptoDataService) StartDataCollection(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isCollecting {
		return fmt.Errorf("data collection is already running")
	}

	s.isCollecting = true
	s.logger.Info("Starting cryptocurrency data collection")

	// Start the collection goroutine
	go s.collectDataLoop(ctx)

	return nil
}

// StopDataCollection stops the background data collection process
func (s *CryptoDataService) StopDataCollection() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isCollecting {
		return
	}

	s.logger.Info("Stopping cryptocurrency data collection")
	close(s.stopChan)
	s.isCollecting = false
	s.stopChan = make(chan struct{})
}

// IsCollecting returns whether data collection is active
func (s *CryptoDataService) IsCollecting() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isCollecting
}

// SetUpdateInterval sets the data collection interval
func (s *CryptoDataService) SetUpdateInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updateInterval = interval
}

// collectDataLoop runs the main data collection loop
func (s *CryptoDataService) collectDataLoop(ctx context.Context) {
	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	// Initial collection
	s.collectAllData(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Data collection stopped due to context cancellation")
			return
		case <-s.stopChan:
			s.logger.Info("Data collection stopped")
			return
		case <-ticker.C:
			s.collectAllData(ctx)
		}
	}
}

// collectAllData collects all cryptocurrency data
func (s *CryptoDataService) collectAllData(ctx context.Context) {
	start := time.Now()
	s.logger.Debug("Starting data collection cycle")

	// Get all active cryptocurrencies from database
	cryptos, err := s.cryptoRepo.GetActive(ctx, 0, 0) // Get all active
	if err != nil {
		s.logger.WithError(err).Error("Failed to get active cryptocurrencies")
		return
	}

	if len(cryptos) == 0 {
		s.logger.Debug("No active cryptocurrencies found, skipping data collection")
		return
	}

	// Collect price data for each cryptocurrency
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent requests

	for _, crypto := range cryptos {
		wg.Add(1)
		go func(crypto entities.CryptoCurrency) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			s.collectCryptoData(ctx, crypto.Symbol)
		}(crypto)
	}

	wg.Wait()

	duration := time.Since(start)
	s.logger.WithFields(logrus.Fields{
		"cryptocurrencies": len(cryptos),
		"duration":         duration,
	}).Debug("Data collection cycle completed")
}

// collectCryptoData collects data for a specific cryptocurrency
func (s *CryptoDataService) collectCryptoData(ctx context.Context, symbol string) {
	// Get current price
	ticker, err := s.binanceClient.GetTickerPrice(ctx, symbol)
	if err != nil {
		s.logger.WithError(err).WithField("symbol", symbol).Error("Failed to get ticker price")
		return
	}

	// Convert price to float
	price, err := strconv.ParseFloat(ticker.Price, 64)
	if err != nil {
		s.logger.WithError(err).WithField("symbol", symbol).Error("Failed to parse price")
		return
	}

	// Store price history
	priceHistory := &entities.PriceHistory{
		Symbol:     symbol,
		Timeframe:  "1m", // 1 minute timeframe for real-time data
		Timestamp:  time.Now(),
		OpenPrice:  price, // For real-time, open = current price
		HighPrice:  price,
		LowPrice:   price,
		ClosePrice: price,
		Volume:     0, // We'll get this from klines later
	}

	if err := s.priceHistoryRepo.Create(ctx, priceHistory); err != nil {
		s.logger.WithError(err).WithField("symbol", symbol).Error("Failed to store price history")
	}

	s.logger.WithFields(logrus.Fields{
		"symbol": symbol,
		"price":  price,
	}).Debug("Collected crypto data")
}

// CollectHistoricalData collects historical data for a symbol
func (s *CryptoDataService) CollectHistoricalData(ctx context.Context, symbol, interval string, limit int) error {
	s.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"interval": interval,
		"limit":    limit,
	}).Info("Collecting historical data")

	// Get klines from Binance
	klines, err := s.binanceClient.GetKlines(ctx, symbol, interval, limit, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to get klines: %w", err)
	}

	// Convert klines to price history entities
	var histories []entities.PriceHistory
	for _, kline := range klines {
		if len(kline) < 6 {
			continue
		}

		// Parse kline data
		openTime := int64(kline[0].(float64))
		open, _ := strconv.ParseFloat(kline[1].(string), 64)
		high, _ := strconv.ParseFloat(kline[2].(string), 64)
		low, _ := strconv.ParseFloat(kline[3].(string), 64)
		close, _ := strconv.ParseFloat(kline[4].(string), 64)
		volume, _ := strconv.ParseFloat(kline[5].(string), 64)

		history := entities.PriceHistory{
			Symbol:     symbol,
			Timeframe:  interval,
			Timestamp:  time.Unix(openTime/1000, 0),
			OpenPrice:  open,
			HighPrice:  high,
			LowPrice:   low,
			ClosePrice: close,
			Volume:     volume,
		}

		histories = append(histories, history)
	}

	// Bulk insert historical data
	if len(histories) > 0 {
		if err := s.priceHistoryRepo.BulkInsert(ctx, histories); err != nil {
			return fmt.Errorf("failed to bulk insert historical data: %w", err)
		}
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":  symbol,
		"records": len(histories),
	}).Info("Historical data collection completed")

	return nil
}

// UpdateCryptocurrencyList updates the list of available cryptocurrencies
func (s *CryptoDataService) UpdateCryptocurrencyList(ctx context.Context) error {
	s.logger.Info("Updating cryptocurrency list")

	// Get exchange info from Binance
	exchangeInfo, err := s.binanceClient.GetExchangeInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get exchange info: %w", err)
	}

	// Filter for active USDT pairs
	var newCryptos []entities.CryptoCurrency
	for _, symbolInfo := range exchangeInfo.Symbols {
		// Only include USDT pairs that are trading
		if symbolInfo.Status == "TRADING" && len(symbolInfo.Symbol) > 4 &&
			symbolInfo.Symbol[len(symbolInfo.Symbol)-4:] == "USDT" {

			// Extract base currency (remove USDT)
			baseCurrency := symbolInfo.Symbol[:len(symbolInfo.Symbol)-4]

			crypto := entities.CryptoCurrency{
				Symbol:     symbolInfo.Symbol,
				Name:       baseCurrency,
				MarketType: "Spot",
				Active:     true,
			}

			newCryptos = append(newCryptos, crypto)
		}
	}

	// Insert new cryptocurrencies (ignore duplicates)
	created := 0
	for _, crypto := range newCryptos {
		// Check if crypto already exists
		existing, err := s.cryptoRepo.GetBySymbol(ctx, crypto.Symbol)
		if err == nil && existing != nil {
			continue // Already exists
		}

		if err := s.cryptoRepo.Create(ctx, &crypto); err != nil {
			s.logger.WithError(err).WithField("symbol", crypto.Symbol).Error("Failed to create cryptocurrency")
			continue
		}
		created++
	}

	s.logger.WithFields(logrus.Fields{
		"total_found": len(newCryptos),
		"created":     created,
	}).Info("Cryptocurrency list update completed")

	return nil
}

// GetCurrentPrice gets the current price for a symbol
func (s *CryptoDataService) GetCurrentPrice(ctx context.Context, symbol string) (float64, error) {
	ticker, err := s.binanceClient.GetTickerPrice(ctx, symbol)
	if err != nil {
		return 0, fmt.Errorf("failed to get current price: %w", err)
	}

	price, err := strconv.ParseFloat(ticker.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price: %w", err)
	}

	return price, nil
}
