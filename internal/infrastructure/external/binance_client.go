package external

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/growthfolio/go-priceguard-api/internal/infrastructure/config"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// BinanceClient handles interactions with Binance API
type BinanceClient struct {
	config     *config.BinanceConfig
	httpClient *http.Client
	baseURL    string
	wsBaseURL  string
	logger     *logrus.Logger

	// Rate limiting
	rateLimiter *rate.Limiter

	// WebSocket
	wsConn      *websocket.Conn
	wsMutex     sync.RWMutex
	wsChannels  map[string]chan []byte
	wsActive    bool
	wsReconnect bool

	// Retry configuration
	maxRetries    int
	retryInterval time.Duration
}

// TickerPrice represents a ticker price from Binance
type TickerPrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

// WebSocketMessage represents a WebSocket message from Binance
type WebSocketMessage struct {
	Stream string      `json:"stream"`
	Data   interface{} `json:"data"`
}

// TickerData represents ticker data from WebSocket
type TickerData struct {
	EventType string `json:"e"`
	EventTime int64  `json:"E"`
	Symbol    string `json:"s"`
	Price     string `json:"c"`
	Change    string `json:"P"`
	Volume    string `json:"v"`
}

// KlineWebSocketData represents kline data from WebSocket
type KlineWebSocketData struct {
	EventType string `json:"e"`
	EventTime int64  `json:"E"`
	Symbol    string `json:"s"`
	Kline     struct {
		Interval  string `json:"i"`
		OpenTime  int64  `json:"t"`
		CloseTime int64  `json:"T"`
		Symbol    string `json:"s"`
		Open      string `json:"o"`
		Close     string `json:"c"`
		High      string `json:"h"`
		Low       string `json:"l"`
		Volume    string `json:"v"`
		IsClosed  bool   `json:"x"`
	} `json:"k"`
}

// KlineData represents a candlestick/kline from Binance
type KlineData struct {
	OpenTime                 int64  `json:"openTime"`
	Open                     string `json:"open"`
	High                     string `json:"high"`
	Low                      string `json:"low"`
	Close                    string `json:"close"`
	Volume                   string `json:"volume"`
	CloseTime                int64  `json:"closeTime"`
	QuoteAssetVolume         string `json:"quoteAssetVolume"`
	NumberOfTrades           int    `json:"numberOfTrades"`
	TakerBuyBaseAssetVolume  string `json:"takerBuyBaseAssetVolume"`
	TakerBuyQuoteAssetVolume string `json:"takerBuyQuoteAssetVolume"`
}

// ExchangeInfo represents exchange information from Binance
type ExchangeInfo struct {
	Timezone   string `json:"timezone"`
	ServerTime int64  `json:"serverTime"`
	Symbols    []struct {
		Symbol string `json:"symbol"`
		Status string `json:"status"`
	} `json:"symbols"`
}

// NewBinanceClient creates a new Binance API client
func NewBinanceClient(cfg *config.BinanceConfig, logger *logrus.Logger) *BinanceClient {
	baseURL := "https://api.binance.com"
	wsBaseURL := "wss://stream.binance.com:9443/ws"
	if cfg.TestNet {
		baseURL = "https://testnet.binance.vision"
		wsBaseURL = "wss://testnet.binance.vision/ws"
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Initialize rate limiter (1200 requests per minute for Binance)
	rateLimiter := rate.NewLimiter(rate.Every(time.Minute/1200), 1200)

	return &BinanceClient{
		config:        cfg,
		httpClient:    client,
		baseURL:       baseURL,
		wsBaseURL:     wsBaseURL,
		logger:        logger,
		rateLimiter:   rateLimiter,
		wsChannels:    make(map[string]chan []byte),
		maxRetries:    3,
		retryInterval: time.Second * 2,
	}
}

// GetTickerPrice gets the current price for a symbol
func (b *BinanceClient) GetTickerPrice(ctx context.Context, symbol string) (*TickerPrice, error) {
	endpoint := "/api/v3/ticker/price"
	params := url.Values{}
	params.Set("symbol", symbol)

	resp, err := b.makeRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticker price: %w", err)
	}
	defer resp.Body.Close()

	var ticker TickerPrice
	if err := json.NewDecoder(resp.Body).Decode(&ticker); err != nil {
		return nil, fmt.Errorf("failed to decode ticker response: %w", err)
	}

	return &ticker, nil
}

// GetAllTickerPrices gets all ticker prices
func (b *BinanceClient) GetAllTickerPrices(ctx context.Context) ([]TickerPrice, error) {
	endpoint := "/api/v3/ticker/price"

	resp, err := b.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get all ticker prices: %w", err)
	}
	defer resp.Body.Close()

	var tickers []TickerPrice
	if err := json.NewDecoder(resp.Body).Decode(&tickers); err != nil {
		return nil, fmt.Errorf("failed to decode tickers response: %w", err)
	}

	return tickers, nil
}

// GetKlines gets candlestick/kline data for a symbol
func (b *BinanceClient) GetKlines(ctx context.Context, symbol, interval string, limit int, startTime, endTime *int64) ([][]interface{}, error) {
	endpoint := "/api/v3/klines"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)

	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if startTime != nil {
		params.Set("startTime", strconv.FormatInt(*startTime, 10))
	}
	if endTime != nil {
		params.Set("endTime", strconv.FormatInt(*endTime, 10))
	}

	resp, err := b.makeRequest(ctx, "GET", endpoint, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}
	defer resp.Body.Close()

	var klines [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&klines); err != nil {
		return nil, fmt.Errorf("failed to decode klines response: %w", err)
	}

	return klines, nil
}

// GetExchangeInfo gets exchange information
func (b *BinanceClient) GetExchangeInfo(ctx context.Context) (*ExchangeInfo, error) {
	endpoint := "/api/v3/exchangeInfo"

	resp, err := b.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get exchange info: %w", err)
	}
	defer resp.Body.Close()

	var info ExchangeInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode exchange info response: %w", err)
	}

	return &info, nil
}

// makeRequest makes an HTTP request to the Binance API
func (b *BinanceClient) makeRequest(ctx context.Context, method, endpoint string, params url.Values) (*http.Response, error) {
	fullURL := b.baseURL + endpoint
	if params != nil {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key if available (for authenticated endpoints)
	if b.config.APIKey != "" {
		req.Header.Set("X-MBX-APIKEY", b.config.APIKey)
	}

	b.logger.WithFields(logrus.Fields{
		"method":   method,
		"url":      fullURL,
		"endpoint": endpoint,
	}).Debug("Making Binance API request")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// HealthCheck checks if the Binance API is accessible
func (b *BinanceClient) HealthCheck(ctx context.Context) error {
	endpoint := "/api/v3/ping"

	resp, err := b.makeRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("binance API health check failed: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// StartWebSocket starts a WebSocket connection for real-time data
func (b *BinanceClient) StartWebSocket(ctx context.Context, streams []string) error {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()

	if b.wsActive {
		return fmt.Errorf("WebSocket connection is already active")
	}

	// Build stream URL
	streamURL := b.wsBaseURL + "/stream?streams=" + fmt.Sprintf("%s", streams[0])
	for _, stream := range streams[1:] {
		streamURL += "/" + stream
	}

	b.logger.WithField("url", streamURL).Info("Connecting to Binance WebSocket")

	// Establish WebSocket connection
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, streamURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	b.wsConn = conn
	b.wsActive = true
	b.wsReconnect = true

	// Start message handling goroutine
	go b.handleWebSocketMessages(ctx)

	return nil
}

// StopWebSocket stops the WebSocket connection
func (b *BinanceClient) StopWebSocket() {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()

	b.wsReconnect = false
	if b.wsConn != nil {
		b.wsConn.Close()
		b.wsConn = nil
	}
	b.wsActive = false

	b.logger.Info("WebSocket connection stopped")
}

// SubscribeToStream subscribes to a specific stream and returns a channel for messages
func (b *BinanceClient) SubscribeToStream(stream string) <-chan []byte {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()

	if _, exists := b.wsChannels[stream]; !exists {
		b.wsChannels[stream] = make(chan []byte, 100)
	}

	return b.wsChannels[stream]
}

// UnsubscribeFromStream unsubscribes from a specific stream
func (b *BinanceClient) UnsubscribeFromStream(stream string) {
	b.wsMutex.Lock()
	defer b.wsMutex.Unlock()

	if ch, exists := b.wsChannels[stream]; exists {
		close(ch)
		delete(b.wsChannels, stream)
	}
}

// handleWebSocketMessages handles incoming WebSocket messages
func (b *BinanceClient) handleWebSocketMessages(ctx context.Context) {
	defer func() {
		b.wsMutex.Lock()
		b.wsActive = false
		b.wsMutex.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if b.wsConn == nil {
				return
			}

			// Read message
			_, message, err := b.wsConn.ReadMessage()
			if err != nil {
				b.logger.WithError(err).Error("Failed to read WebSocket message")

				// Attempt reconnection if enabled
				if b.wsReconnect {
					b.logger.Info("Attempting to reconnect WebSocket")
					time.Sleep(b.retryInterval)
					// Implement reconnection logic here if needed
				}
				return
			}

			// Parse and route message
			var wsMsg WebSocketMessage
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				b.logger.WithError(err).Error("Failed to parse WebSocket message")
				continue
			}

			// Route message to appropriate channel
			b.wsMutex.RLock()
			if ch, exists := b.wsChannels[wsMsg.Stream]; exists {
				select {
				case ch <- message:
				default:
					// Channel is full, drop message
					b.logger.Warn("WebSocket channel is full, dropping message")
				}
			}
			b.wsMutex.RUnlock()
		}
	}
}

// GetTickerWebSocketStream returns the stream name for ticker data
func (b *BinanceClient) GetTickerWebSocketStream(symbol string) string {
	return fmt.Sprintf("%s@ticker", strings.ToLower(symbol))
}

// GetKlineWebSocketStream returns the stream name for kline data
func (b *BinanceClient) GetKlineWebSocketStream(symbol, interval string) string {
	return fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), interval)
}

// makeRequestWithRetry makes an HTTP request with retry logic
func (b *BinanceClient) makeRequestWithRetry(ctx context.Context, method, endpoint string, params url.Values) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= b.maxRetries; attempt++ {
		// Apply rate limiting
		if err := b.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter error: %w", err)
		}

		resp, err := b.makeRequest(ctx, method, endpoint, params)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		// Don't retry on client errors (4xx)
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return resp, err
		}

		if attempt < b.maxRetries {
			backoff := time.Duration(attempt+1) * b.retryInterval
			b.logger.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"backoff": backoff,
				"error":   err,
			}).Warn("Request failed, retrying")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", b.maxRetries+1, lastErr)
}
