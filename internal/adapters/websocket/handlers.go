package websocket

import (
	"time"

	"github.com/felipe-macedo/go-priceguard-api/internal/application/services"
	"github.com/felipe-macedo/go-priceguard-api/internal/domain/entities"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// WebSocketHandler handles WebSocket related operations
type WebSocketHandler struct {
	hub                       *Hub
	cryptoDataService         *services.CryptoDataService
	technicalIndicatorService *services.TechnicalIndicatorService
	pullbackEntryService      *services.PullbackEntryService
	logger                    *logrus.Logger
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(
	hub *Hub,
	cryptoDataService *services.CryptoDataService,
	technicalIndicatorService *services.TechnicalIndicatorService,
	pullbackEntryService *services.PullbackEntryService,
	logger *logrus.Logger,
) *WebSocketHandler {
	return &WebSocketHandler{
		hub:                       hub,
		cryptoDataService:         cryptoDataService,
		technicalIndicatorService: technicalIndicatorService,
		pullbackEntryService:      pullbackEntryService,
		logger:                    logger,
	}
}

// HandleConnection handles WebSocket connection upgrade
func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	h.hub.HandleWebSocket(c)
}

// BroadcastCryptoDataUpdate broadcasts crypto data updates to subscribed clients
func (h *WebSocketHandler) BroadcastCryptoDataUpdate(symbol string, priceData *entities.PriceHistory) {
	room := "crypto_" + symbol

	// Prepare the data to broadcast
	data := map[string]interface{}{
		"symbol":      symbol,
		"open_price":  priceData.OpenPrice,
		"high_price":  priceData.HighPrice,
		"low_price":   priceData.LowPrice,
		"close_price": priceData.ClosePrice,
		"volume":      priceData.Volume,
		"timestamp":   priceData.Timestamp,
		"timeframe":   priceData.Timeframe,
	}

	// Broadcast to room
	h.hub.Broadcast(room, "crypto_data_update", data)

	h.logger.WithFields(logrus.Fields{
		"symbol":      symbol,
		"close_price": priceData.ClosePrice,
		"room":        room,
	}).Debug("Broadcasted crypto data update")
}

// BroadcastAlertTriggered broadcasts alert triggered events to specific users
func (h *WebSocketHandler) BroadcastAlertTriggered(alert *entities.Alert, currentPrice float64) {
	// Send to specific user
	data := map[string]interface{}{
		"alert_id":       alert.ID,
		"symbol":         alert.Symbol,
		"alert_type":     alert.AlertType,
		"condition_type": alert.ConditionType,
		"target_value":   alert.TargetValue,
		"current_price":  currentPrice,
		"triggered_at":   time.Now(),
		"timeframe":      alert.Timeframe,
	}

	h.hub.BroadcastToUser(alert.UserID, "alert_triggered", data)

	h.logger.WithFields(logrus.Fields{
		"alert_id":      alert.ID,
		"user_id":       alert.UserID,
		"symbol":        alert.Symbol,
		"current_price": currentPrice,
	}).Info("Broadcasted alert triggered")
}

// BroadcastTechnicalIndicatorUpdate broadcasts technical indicator updates
func (h *WebSocketHandler) BroadcastTechnicalIndicatorUpdate(symbol string, indicators map[string]float64) {
	room := "indicators_" + symbol

	data := map[string]interface{}{
		"symbol":     symbol,
		"indicators": indicators,
		"timestamp":  time.Now(),
	}

	h.hub.Broadcast(room, "technical_indicator_update", data)

	h.logger.WithFields(logrus.Fields{
		"symbol": symbol,
		"room":   room,
	}).Debug("Broadcasted technical indicator update")
}

// BroadcastPullbackSignal broadcasts pullback entry signals
func (h *WebSocketHandler) BroadcastPullbackSignal(symbol string, signal map[string]interface{}) {
	room := "pullback_" + symbol

	data := map[string]interface{}{
		"symbol":    symbol,
		"signal":    signal,
		"timestamp": time.Now(),
	}

	h.hub.Broadcast(room, "pullback_signal", data)

	h.logger.WithFields(logrus.Fields{
		"symbol": symbol,
		"room":   room,
	}).Info("Broadcasted pullback signal")
}

// BroadcastMarketSummary broadcasts market summary data
func (h *WebSocketHandler) BroadcastMarketSummary(summary map[string]interface{}) {
	room := "market_summary"

	data := map[string]interface{}{
		"summary":   summary,
		"timestamp": time.Now(),
	}

	h.hub.Broadcast(room, "market_summary_update", data)

	h.logger.WithField("room", room).Debug("Broadcasted market summary")
}

// GetConnectionStats returns WebSocket connection statistics
func (h *WebSocketHandler) GetConnectionStats() map[string]interface{} {
	return map[string]interface{}{
		"connected_clients": h.hub.GetConnectedClients(),
		"active_rooms":      h.hub.GetRooms(),
		"timestamp":         time.Now(),
	}
}

// CleanupInactiveConnections removes inactive connections
func (h *WebSocketHandler) CleanupInactiveConnections() {
	// This could be called periodically to clean up stale connections
	// Implementation would check LastSeen timestamps and close inactive connections
	h.logger.Info("Cleaning up inactive WebSocket connections")
}
