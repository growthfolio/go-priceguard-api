package websocket

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/growthfolio/go-priceguard-api/internal/domain/entities"
	"github.com/sirupsen/logrus"
)

// Hub manages WebSocket connections and broadcasting
// AuthService defines the token validation behavior required by the hub
type AuthService interface {
	ValidateToken(ctx context.Context, accessToken string) (*entities.User, error)
}

type Hub struct {
	clients     map[string]*Client     // Connected clients
	rooms       map[string]*Room       // Chat rooms/channels
	register    chan *Client           // Register requests from clients
	unregister  chan *Client           // Unregister requests from clients
	broadcast   chan *BroadcastMessage // Broadcast messages
	authService AuthService
	logger      *logrus.Logger
	mutex       sync.RWMutex
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// Client represents a WebSocket client
type Client struct {
	ID       string          `json:"id"`
	UserID   uuid.UUID       `json:"user_id"`
	Conn     *websocket.Conn `json:"-"`
	Hub      *Hub            `json:"-"`
	Send     chan []byte     `json:"-"`
	Rooms    map[string]bool `json:"rooms"`
	LastSeen time.Time       `json:"last_seen"`
	mutex    sync.RWMutex
}

// Room represents a WebSocket room/channel
type Room struct {
	ID      string             `json:"id"`
	Clients map[string]*Client `json:"-"`
	mutex   sync.RWMutex
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	Room   string      `json:"room"`
	Type   string      `json:"type"`
	Data   interface{} `json:"data"`
	UserID *uuid.UUID  `json:"user_id,omitempty"`
}

// WebSocketMessage represents incoming/outgoing WebSocket messages
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// SubscribeMessage represents subscription messages
type SubscribeMessage struct {
	Room   string `json:"room"`
	Symbol string `json:"symbol,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, implement proper origin checking
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// NewHub creates a new WebSocket hub
func NewHub(authService AuthService, logger *logrus.Logger) *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		rooms:       make(map[string]*Room),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *BroadcastMessage, 256),
		authService: authService,
		logger:      logger,
		stopChan:    make(chan struct{}),
	}
}

// Start starts the WebSocket hub
func (h *Hub) Start() {
	h.logger.Info("Starting WebSocket hub")
	h.wg.Add(1)

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToRoom(message)

		case <-h.stopChan:
			h.logger.Info("Stopping WebSocket hub")
			h.wg.Done()
			return
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client.ID] = client

	h.logger.WithFields(logrus.Fields{
		"client_id": client.ID,
		"user_id":   client.UserID,
	}).Info("Client registered")

	// Send welcome message
	welcomeMsg := WebSocketMessage{
		Type: "welcome",
		Data: map[string]interface{}{
			"client_id": client.ID,
			"timestamp": time.Now(),
		},
	}
	client.SendMessage(welcomeMsg)
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, exists := h.clients[client.ID]; exists {
		// Remove from all rooms
		for roomID := range client.Rooms {
			h.leaveRoom(client, roomID)
		}

		delete(h.clients, client.ID)
		close(client.Send)

		h.logger.WithFields(logrus.Fields{
			"client_id": client.ID,
			"user_id":   client.UserID,
		}).Info("Client unregistered")
	}
}

// broadcastToRoom broadcasts a message to all clients in a room
func (h *Hub) broadcastToRoom(message *BroadcastMessage) {
	h.mutex.RLock()
	room, exists := h.rooms[message.Room]
	h.mutex.RUnlock()

	if !exists {
		return
	}

	room.mutex.RLock()
	defer room.mutex.RUnlock()

	wsMessage := WebSocketMessage{
		Type: message.Type,
		Data: message.Data,
	}

	messageBytes, err := json.Marshal(wsMessage)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal broadcast message")
		return
	}

	for _, client := range room.Clients {
		select {
		case client.Send <- messageBytes:
		default:
			// Client's send channel is full, remove client
			h.unregister <- client
		}
	}
}

// joinRoom adds a client to a room
func (h *Hub) joinRoom(client *Client, roomID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Create room if it doesn't exist
	if _, exists := h.rooms[roomID]; !exists {
		h.rooms[roomID] = &Room{
			ID:      roomID,
			Clients: make(map[string]*Client),
		}
	}

	room := h.rooms[roomID]

	room.mutex.Lock()
	room.Clients[client.ID] = client
	room.mutex.Unlock()

	client.mutex.Lock()
	client.Rooms[roomID] = true
	client.mutex.Unlock()

	h.logger.WithFields(logrus.Fields{
		"client_id": client.ID,
		"room_id":   roomID,
	}).Info("Client joined room")
}

// leaveRoom removes a client from a room
func (h *Hub) leaveRoom(client *Client, roomID string) {
	room, exists := h.rooms[roomID]
	if !exists {
		return
	}

	room.mutex.Lock()
	delete(room.Clients, client.ID)
	isEmpty := len(room.Clients) == 0
	room.mutex.Unlock()

	client.mutex.Lock()
	delete(client.Rooms, roomID)
	client.mutex.Unlock()

	// Remove empty room
	if isEmpty {
		delete(h.rooms, roomID)
	}

	h.logger.WithFields(logrus.Fields{
		"client_id": client.ID,
		"room_id":   roomID,
	}).Info("Client left room")
}

// HandleWebSocket handles WebSocket upgrade and client management
func (h *Hub) HandleWebSocket(c *gin.Context) {
	// Get JWT token from query parameter
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
		return
	}

	// Verify token
	user, err := h.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Create client
	client := &Client{
		ID:       uuid.New().String(),
		UserID:   user.ID,
		Conn:     conn,
		Hub:      h,
		Send:     make(chan []byte, 256),
		Rooms:    make(map[string]bool),
		LastSeen: time.Now(),
	}

	// Register client
	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// Broadcast sends a message to a specific room
func (h *Hub) Broadcast(room, messageType string, data interface{}) {
	h.broadcast <- &BroadcastMessage{
		Room: room,
		Type: messageType,
		Data: data,
	}
}

// BroadcastToUser sends a message to a specific user
func (h *Hub) BroadcastToUser(userID uuid.UUID, messageType string, data interface{}) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for _, client := range h.clients {
		if client.UserID == userID {
			wsMessage := WebSocketMessage{
				Type: messageType,
				Data: data,
			}
			client.SendMessage(wsMessage)
		}
	}
}

// GetConnectedClients returns the number of connected clients
func (h *Hub) GetConnectedClients() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetRooms returns information about active rooms
func (h *Hub) GetRooms() map[string]int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	rooms := make(map[string]int)
	for roomID, room := range h.rooms {
		room.mutex.RLock()
		rooms[roomID] = len(room.Clients)
		room.mutex.RUnlock()
	}

	return rooms
}

// Stop gracefully stops the hub goroutine
func (h *Hub) Stop() {
	close(h.stopChan)
	h.wg.Wait()
}
