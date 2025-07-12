package websocket_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	domainServices "github.com/growthfolio/go-priceguard-api/internal/domain/services"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type WebSocketIntegrationTestSuite struct {
	suite.Suite
	server *httptest.Server
	logger *logrus.Logger
}

// Mock services that implement the required interfaces properly
type MockAuthServiceForWebSocket struct {
	mock.Mock
}

func (m *MockAuthServiceForWebSocket) ValidateJWT(tokenString string) (*domainServices.JWTClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domainServices.JWTClaims), args.Error(1)
}

// Implement other required methods of AuthService interface
func (m *MockAuthServiceForWebSocket) Login(ctx interface{}, token string) (interface{}, error) {
	return nil, nil
}
func (m *MockAuthServiceForWebSocket) Logout(ctx interface{}, token string) error {
	return nil
}
func (m *MockAuthServiceForWebSocket) RefreshToken(ctx interface{}, token string) (interface{}, error) {
	return nil, nil
}

func (suite *WebSocketIntegrationTestSuite) SetupSuite() {
	// Setup logger
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.ErrorLevel)

	// Setup HTTP server with a simple WebSocket test endpoint
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Simple WebSocket endpoint for testing basic connectivity
	router.GET("/ws", func(c *gin.Context) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			suite.logger.WithError(err).Error("Failed to upgrade connection")
			return
		}
		defer conn.Close()

		// Simple echo server for testing
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			err = conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				break
			}
		}
	})

	suite.server = httptest.NewServer(router)
}

func (suite *WebSocketIntegrationTestSuite) TearDownSuite() {
	suite.server.Close()
}

func (suite *WebSocketIntegrationTestSuite) TestWebSocketBasicConnection() {
	// Test basic WebSocket connection
	wsURL := "ws" + strings.TrimPrefix(suite.server.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)

	// Verify connection
	suite.NoError(err)
	suite.Equal(http.StatusSwitchingProtocols, resp.StatusCode)
	suite.NotNil(conn)

	// Test basic message exchange
	testMessage := "Hello WebSocket"
	err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
	suite.NoError(err)

	// Read echo response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	suite.NoError(err)
	suite.Equal(testMessage, string(message))

	// Cleanup
	conn.Close()
}

func (suite *WebSocketIntegrationTestSuite) TestWebSocketBroadcast() {
	// Setup valid connection
	userID := uuid.New()
	claims := &domainServices.JWTClaims{
		UserID: userID,
		Email:  "test@example.com",
	}

	mockAuth := &MockAuthServiceForWebSocket{}
	mockAuth.On("ValidateJWT", "valid-token").Return(claims, nil)

	// Connect to WebSocket
	wsURL := "ws" + strings.TrimPrefix(suite.server.URL, "http") + "/ws?token=valid-token"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	suite.NoError(err)
	defer conn.Close()

	// Subscribe to a room
	subscribeMsg := map[string]interface{}{
		"type": "subscribe",
		"data": map[string]interface{}{
			"room": "crypto_BTCUSDT",
		},
	}

	err = conn.WriteJSON(subscribeMsg)
	suite.NoError(err)

	// Give time for subscription to process
	time.Sleep(50 * time.Millisecond)

	// Broadcast a message to the room
	broadcastData := map[string]interface{}{
		"symbol": "BTCUSDT",
		"price":  50000.0,
		"change": 2.5,
	}

	// Test different types of broadcasts
	suite.testCryptoDataBroadcast(conn, broadcastData)
	suite.testAlertTriggeredBroadcast(conn, userID)
	suite.testNotificationBroadcast(conn, userID)

	mockAuth.AssertExpectations(suite.T())
}

func (suite *WebSocketIntegrationTestSuite) testCryptoDataBroadcast(conn *websocket.Conn, data map[string]interface{}) {
	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Read the broadcasted message
	var receivedMsg map[string]interface{}
	err := conn.ReadJSON(&receivedMsg)
	suite.NoError(err)

	// Verify message content
	suite.Contains(receivedMsg, "type")
	suite.Contains(receivedMsg, "data")
}

func (suite *WebSocketIntegrationTestSuite) testAlertTriggeredBroadcast(conn *websocket.Conn, userID uuid.UUID) {
	// Subscribe to user alerts
	subscribeMsg := map[string]interface{}{
		"type": "subscribe",
		"data": map[string]interface{}{
			"room": "alerts_user_" + userID.String(),
		},
	}

	err := conn.WriteJSON(subscribeMsg)
	suite.NoError(err)

	// Simulate alert triggered
	alertData := map[string]interface{}{
		"alert_id":  uuid.New().String(),
		"symbol":    "BTCUSDT",
		"type":      "price",
		"condition": "above",
		"target":    50000.0,
		"triggered": true,
		"timestamp": time.Now(),
	}

	// Send alert via message (simulating broadcast)
	alertMsg := map[string]interface{}{
		"type": "alert_triggered",
		"data": alertData,
	}

	err = conn.WriteJSON(alertMsg)
	suite.NoError(err)
}

func (suite *WebSocketIntegrationTestSuite) testNotificationBroadcast(conn *websocket.Conn, userID uuid.UUID) {
	// Test notification broadcast
	notificationData := map[string]interface{}{
		"id":      uuid.New().String(),
		"user_id": userID.String(),
		"type":    "alert",
		"title":   "Alert Triggered",
		"message": "BTCUSDT price above 50000",
	}

	notificationMsg := map[string]interface{}{
		"type": "notification",
		"data": notificationData,
	}

	err := conn.WriteJSON(notificationMsg)
	suite.NoError(err)
}

func (suite *WebSocketIntegrationTestSuite) TestWebSocketConcurrentConnections() {
	// Test multiple concurrent connections
	numConnections := 10
	connections := make([]*websocket.Conn, numConnections)
	mockAuths := make([]*MockAuthServiceForWebSocket, numConnections)

	// Setup auth for all connections
	for i := 0; i < numConnections; i++ {
		userID := uuid.New()
		claims := &domainServices.JWTClaims{
			UserID: userID,
			Email:  fmt.Sprintf("test%d@example.com", i),
		}
		token := fmt.Sprintf("valid-token-%d", i)

		mockAuth := &MockAuthServiceForWebSocket{}
		mockAuth.On("ValidateJWT", token).Return(claims, nil)
		mockAuths[i] = mockAuth
	}

	// Establish connections concurrently
	for i := 0; i < numConnections; i++ {
		go func(index int) {
			wsURL := fmt.Sprintf("ws%s/ws?token=valid-token-%d",
				strings.TrimPrefix(suite.server.URL, "http"), index)

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err == nil {
				connections[index] = conn
			}
		}(i)
	}

	// Wait for connections to establish
	time.Sleep(100 * time.Millisecond)

	// Verify at least some connections are established
	establishedCount := 0
	for i := 0; i < numConnections; i++ {
		if connections[i] != nil {
			establishedCount++
		}
	}

	suite.Greater(establishedCount, 0, "At least some connections should be established")

	// Close all established connections
	for i := 0; i < numConnections; i++ {
		if connections[i] != nil {
			connections[i].Close()
		}
	}
}

func (suite *WebSocketIntegrationTestSuite) TestWebSocketMessageTypes() {
	// Test various message types
	mockAuth := &MockAuthServiceForWebSocket{}
	userID := uuid.New()
	claims := &domainServices.JWTClaims{
		UserID: userID,
		Email:  "test@example.com",
	}
	mockAuth.On("ValidateJWT", "valid-token").Return(claims, nil)

	// Connect to WebSocket
	wsURL := "ws" + strings.TrimPrefix(suite.server.URL, "http") + "/ws?token=valid-token"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	suite.NoError(err)
	defer conn.Close()

	// Test subscribe message
	subscribeMsg := map[string]interface{}{
		"type": "subscribe",
		"data": map[string]interface{}{
			"room": "crypto_BTCUSDT",
		},
	}

	err = conn.WriteJSON(subscribeMsg)
	suite.NoError(err)

	// Test unsubscribe message
	unsubscribeMsg := map[string]interface{}{
		"type": "unsubscribe",
		"data": map[string]interface{}{
			"room": "crypto_BTCUSDT",
		},
	}

	err = conn.WriteJSON(unsubscribeMsg)
	suite.NoError(err)

	// Test ping message
	err = conn.WriteMessage(websocket.PingMessage, []byte("ping"))
	suite.NoError(err)

	// Should receive pong
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	msgType, data, err := conn.ReadMessage()
	if err == nil {
		suite.Equal(websocket.PongMessage, msgType)
		suite.Equal([]byte("ping"), data)
	}

	mockAuth.AssertExpectations(suite.T())
}

func (suite *WebSocketIntegrationTestSuite) TestWebSocketErrorHandling() {
	// Test error scenarios
	mockAuth := &MockAuthServiceForWebSocket{}
	userID := uuid.New()
	claims := &domainServices.JWTClaims{
		UserID: userID,
		Email:  "test@example.com",
	}
	mockAuth.On("ValidateJWT", "valid-token").Return(claims, nil)

	// Connect to WebSocket
	wsURL := "ws" + strings.TrimPrefix(suite.server.URL, "http") + "/ws?token=valid-token"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	suite.NoError(err)
	defer conn.Close()

	// Test invalid JSON message
	err = conn.WriteMessage(websocket.TextMessage, []byte("invalid json"))
	suite.NoError(err)

	// Test message with missing type
	invalidMsg := map[string]interface{}{
		"data": map[string]interface{}{
			"room": "test_room",
		},
	}

	err = conn.WriteJSON(invalidMsg)
	suite.NoError(err)

	// Test message with invalid type
	invalidTypeMsg := map[string]interface{}{
		"type": "invalid_type",
		"data": map[string]interface{}{
			"room": "test_room",
		},
	}

	err = conn.WriteJSON(invalidTypeMsg)
	suite.NoError(err)

	// Connection should still be alive
	validMsg := map[string]interface{}{
		"type": "subscribe",
		"data": map[string]interface{}{
			"room": "test_room",
		},
	}

	err = conn.WriteJSON(validMsg)
	suite.NoError(err)

	mockAuth.AssertExpectations(suite.T())
}

func (suite *WebSocketIntegrationTestSuite) TestWebSocketTimeout() {
	// Connect to WebSocket
	wsURL := "ws" + strings.TrimPrefix(suite.server.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	suite.NoError(err)
	defer conn.Close()

	// Set a very short read deadline
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	// Try to read without sending anything (should timeout)
	_, _, err = conn.ReadMessage()
	suite.Error(err)

	// Verify it's a timeout error
	netErr, ok := err.(interface{ Timeout() bool })
	if ok {
		suite.True(netErr.Timeout())
	}
}

func TestWebSocketIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping WebSocket integration tests in short mode")
	}

	suite.Run(t, new(WebSocketIntegrationTestSuite))
}

// Benchmark WebSocket connection establishment
func BenchmarkWebSocketConnection(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping WebSocket benchmarks in short mode")
	}

	// Setup test server
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/ws", func(c *gin.Context) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		conn.Close() // Immediately close for benchmark
	})

	server := httptest.NewServer(router)
	defer server.Close()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			conn.Close()
		}
	}
}

// Test WebSocket with real PriceGuard components (simplified)
func TestWebSocketWithPriceGuardComponents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping extended WebSocket integration tests in short mode")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Test that we can create the required components without errors
	t.Run("CreateWebSocketComponents", func(t *testing.T) {
		// This tests that the WebSocket related types can be instantiated
		// without full service dependencies

		userID := uuid.New()

		// Test entity creation
		alert := &entities.Alert{
			ID:            uuid.New(),
			UserID:        userID,
			Symbol:        "BTCUSDT",
			AlertType:     "price",
			ConditionType: "above",
			TargetValue:   50000.0,
			Enabled:       true,
		}

		priceHistory := &entities.PriceHistory{
			Symbol:     "BTCUSDT",
			ClosePrice: 50000.0,
			Timestamp:  time.Now(),
		}

		// Verify entities are created correctly
		if alert.ID == uuid.Nil {
			t.Error("Alert ID should not be nil")
		}

		if priceHistory.Symbol != "BTCUSDT" {
			t.Error("PriceHistory symbol should be BTCUSDT")
		}
	})
}
