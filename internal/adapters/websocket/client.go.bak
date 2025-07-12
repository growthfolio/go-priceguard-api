package websocket

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.WithError(err).Error("WebSocket unexpected close error")
			}
			break
		}

		// Update last seen
		c.mutex.Lock()
		c.LastSeen = time.Now()
		c.mutex.Unlock()

		// Parse message
		var msg WebSocketMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			c.Hub.logger.WithError(err).Error("Failed to parse WebSocket message")
			continue
		}

		// Handle message
		c.handleMessage(msg)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(message WebSocketMessage) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		c.Hub.logger.WithError(err).Error("Failed to marshal message")
		return
	}

	select {
	case c.Send <- messageBytes:
	default:
		close(c.Send)
		delete(c.Hub.clients, c.ID)
	}
}

// handleMessage handles incoming messages from the client
func (c *Client) handleMessage(msg WebSocketMessage) {
	switch msg.Type {
	case "subscribe":
		c.handleSubscribe(msg)
	case "unsubscribe":
		c.handleUnsubscribe(msg)
	case "ping":
		c.handlePing()
	default:
		c.Hub.logger.WithField("type", msg.Type).Warn("Unknown message type")
	}
}

// handleSubscribe handles subscription messages
func (c *Client) handleSubscribe(msg WebSocketMessage) {
	var subMsg SubscribeMessage

	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		c.Hub.logger.WithError(err).Error("Failed to marshal subscription data")
		return
	}

	if err := json.Unmarshal(dataBytes, &subMsg); err != nil {
		c.Hub.logger.WithError(err).Error("Failed to parse subscription message")
		return
	}

	// Join the requested room
	c.Hub.joinRoom(c, subMsg.Room)

	// Send confirmation
	response := WebSocketMessage{
		Type: "subscribed",
		Data: map[string]interface{}{
			"room":   subMsg.Room,
			"symbol": subMsg.Symbol,
		},
	}
	c.SendMessage(response)
}

// handleUnsubscribe handles unsubscription messages
func (c *Client) handleUnsubscribe(msg WebSocketMessage) {
	var subMsg SubscribeMessage

	dataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		c.Hub.logger.WithError(err).Error("Failed to marshal unsubscription data")
		return
	}

	if err := json.Unmarshal(dataBytes, &subMsg); err != nil {
		c.Hub.logger.WithError(err).Error("Failed to parse unsubscription message")
		return
	}

	// Leave the room
	c.Hub.leaveRoom(c, subMsg.Room)

	// Send confirmation
	response := WebSocketMessage{
		Type: "unsubscribed",
		Data: map[string]interface{}{
			"room": subMsg.Room,
		},
	}
	c.SendMessage(response)
}

// handlePing handles ping messages
func (c *Client) handlePing() {
	response := WebSocketMessage{
		Type: "pong",
		Data: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}
	c.SendMessage(response)
}
