package websocket_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gws "github.com/gorilla/websocket"
	ws "github.com/growthfolio/go-priceguard-api/internal/adapters/websocket"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockAuthService for testing
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) ValidateToken(ctx context.Context, tokenString string) (*entities.User, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

type WebSocketTestSuite struct {
	suite.Suite
	hub      *ws.Hub
	handler  *ws.WebSocketHandler
	mockAuth *MockAuthService
	logger   *logrus.Logger
	ctx      context.Context
}

func (suite *WebSocketTestSuite) SetupTest() {
	suite.mockAuth = &MockAuthService{}
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	suite.ctx = context.Background()

	suite.hub = ws.NewHub(suite.mockAuth, suite.logger)
	suite.handler = ws.NewWebSocketHandler(
		suite.hub,
		nil,
		nil,
		nil,
		suite.logger,
	)

	// Start the hub
	go suite.hub.Start()

	// Give the hub time to start
	time.Sleep(10 * time.Millisecond)
}

func (suite *WebSocketTestSuite) TearDownTest() {
	// Stop the hub
	suite.hub.Stop()

	// Assert mock expectations
	suite.mockAuth.AssertExpectations(suite.T())
}

func (suite *WebSocketTestSuite) TestNewHub() {
	// Test that NewHub creates a valid instance
	assert.NotNil(suite.T(), suite.hub)
}

func (suite *WebSocketTestSuite) TestNewWebSocketHandler() {
	// Test that NewWebSocketHandler creates a valid instance
	assert.NotNil(suite.T(), suite.handler)
}

func (suite *WebSocketTestSuite) TestBroadcastCryptoDataUpdate() {
	// Test broadcasting crypto data updates
	symbol := "BTCUSDT"
	priceData := &entities.PriceHistory{
		Symbol:     symbol,
		ClosePrice: 50000.0,
		Timestamp:  time.Now(),
	}

	// This should not panic or error
	assert.NotPanics(suite.T(), func() {
		suite.handler.BroadcastCryptoDataUpdate(symbol, priceData)
	})
}

func (suite *WebSocketTestSuite) TestBroadcastAlertTriggered() {
	// Test broadcasting alert triggered events
	alert := &entities.Alert{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		Symbol:        "BTCUSDT",
		AlertType:     "price",
		ConditionType: "above",
		TargetValue:   50000.0,
	}
	currentPrice := 51000.0

	// This should not panic or error
	assert.NotPanics(suite.T(), func() {
		suite.handler.BroadcastAlertTriggered(alert, currentPrice)
	})
}

func (suite *WebSocketTestSuite) TestBroadcastTechnicalIndicatorUpdate() {
	// Test broadcasting technical indicator updates
	symbol := "BTCUSDT"
	indicators := map[string]float64{
		"rsi":    65.5,
		"ema_12": 50500.0,
		"ema_26": 50200.0,
		"macd":   150.0,
	}

	// This should not panic or error
	assert.NotPanics(suite.T(), func() {
		suite.handler.BroadcastTechnicalIndicatorUpdate(symbol, indicators)
	})
}

func (suite *WebSocketTestSuite) TestBroadcastPullbackSignal() {
	// Test broadcasting pullback signals
	symbol := "BTCUSDT"
	signal := map[string]interface{}{
		"signal":     "buy",
		"confidence": 0.85,
		"price":      50000.0,
		"timestamp":  time.Now(),
	}

	// This should not panic or error
	assert.NotPanics(suite.T(), func() {
		suite.handler.BroadcastPullbackSignal(symbol, signal)
	})
}

func (suite *WebSocketTestSuite) TestBroadcastMarketSummary() {
	// Test broadcasting market summary
	summary := map[string]interface{}{
		"total_volume":      1000000000,
		"top_gainers":       []string{"BTCUSDT", "ETHUSDT"},
		"top_losers":        []string{"ADAUSDT"},
		"market_cap_change": 2.5,
	}

	// This should not panic or error
	assert.NotPanics(suite.T(), func() {
		suite.handler.BroadcastMarketSummary(summary)
	})
}

func (suite *WebSocketTestSuite) TestGetConnectionStats() {
	// Test getting connection statistics
	stats := suite.handler.GetConnectionStats()

	assert.NotNil(suite.T(), stats)
	assert.Contains(suite.T(), stats, "connected_clients")
	assert.Contains(suite.T(), stats, "active_rooms")
	assert.Contains(suite.T(), stats, "timestamp")
}

func (suite *WebSocketTestSuite) TestCleanupInactiveConnections() {
	// Test cleanup of inactive connections
	// This should not panic
	assert.NotPanics(suite.T(), func() {
		suite.handler.CleanupInactiveConnections()
	})
}

func (suite *WebSocketTestSuite) TestHubBroadcast() {
	// Test hub broadcast functionality
	room := "test_room"
	messageType := "test_message"
	data := map[string]interface{}{
		"message":   "Hello, WebSocket!",
		"timestamp": time.Now(),
	}

	// This should not panic
	assert.NotPanics(suite.T(), func() {
		suite.hub.Broadcast(room, messageType, data)
	})
}

func (suite *WebSocketTestSuite) TestHubBroadcastToUser() {
	// Test hub broadcast to specific user
	userID := uuid.New()
	messageType := "user_message"
	data := map[string]interface{}{
		"message":   "Hello, User!",
		"timestamp": time.Now(),
	}

	// This should not panic
	assert.NotPanics(suite.T(), func() {
		suite.hub.BroadcastToUser(userID, messageType, data)
	})
}

func (suite *WebSocketTestSuite) TestHubGetConnectedClients() {
	// Test getting connected clients count
	count := suite.hub.GetConnectedClients()
	assert.GreaterOrEqual(suite.T(), count, 0)
}

func (suite *WebSocketTestSuite) TestHubGetRooms() {
	// Test getting rooms information
	rooms := suite.hub.GetRooms()
	assert.NotNil(suite.T(), rooms)
}

func TestWebSocketTestSuite(t *testing.T) {
	suite.Run(t, new(WebSocketTestSuite))
}

// Additional unit tests for WebSocket functionality
func TestWebSocketHandler_HandleConnection_NoAuth(t *testing.T) {
	mockAuth := &MockAuthService{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	hub := ws.NewHub(mockAuth, logger)
	handler := ws.NewWebSocketHandler(hub, nil, nil, nil, logger)

	// Start hub
	go hub.Start()
	defer hub.Stop()

	// Create test server using gin router
	router := gin.New()
	router.GET("/ws", handler.HandleConnection)
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.1
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Try to connect without token (should fail or be rejected)
	_, _, err := gws.DefaultDialer.Dial(wsURL, nil)

	// Connection should fail due to missing authentication
	assert.Error(t, err)
}

func TestWebSocketHandler_HandleConnection_InvalidAuth(t *testing.T) {
	mockAuth := &MockAuthService{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	hub := ws.NewHub(mockAuth, logger)
	handler := ws.NewWebSocketHandler(hub, nil, nil, nil, logger)

	// Start hub
	go hub.Start()
	defer hub.Stop()

	// Mock invalid token validation
	mockAuth.On("ValidateToken", "invalid_token").Return((*entities.User)(nil), assert.AnError)

	// Create test server using gin router
	router := gin.New()
	router.GET("/ws", handler.HandleConnection)
	server := httptest.NewServer(router)
	defer server.Close()

	// Convert http://127.0.0.1 to ws://127.0.0.1
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=invalid_token"

	// Try to connect with invalid token
	_, _, err := gws.DefaultDialer.Dial(wsURL, nil)

	// Connection should fail due to invalid token
	assert.Error(t, err)

	mockAuth.AssertExpectations(t)
}

func TestWebSocketWorker_StartStop(t *testing.T) {
	mockAuth := &MockAuthService{}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	hub := ws.NewHub(mockAuth, logger)
	handler := ws.NewWebSocketHandler(hub, nil, nil, nil, logger)

	// Create worker with nil services for basic testing
	worker := ws.NewWorker(
		hub,
		handler,
		nil, // CryptoDataService
		nil, // TechnicalIndicatorService
		nil, // PullbackEntryService
		nil, // AlertEngine
		nil, // NotificationService
		nil, // AlertRepository
		nil, // PriceHistoryRepository
		logger,
	)

	// Start hub and worker
	go hub.Start()
	defer hub.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker
	worker.Start(ctx)

	// Give worker time to start
	time.Sleep(10 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// Test should complete without hanging
	assert.True(t, true) // Placeholder assertion
}
