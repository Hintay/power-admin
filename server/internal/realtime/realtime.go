package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Config represents realtime service configuration
type Config struct {
	EnableWebSocket bool
	EnableSSE       bool
	MaxConnections  int
}

// Hub maintains active connections and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// Client represents a connected client
type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	userID     uint
	clientID   string
	clientType string // "admin", "user", "collector"
}

// Message represents a realtime message
type Message struct {
	Type        string      `json:"type"`
	Event       string      `json:"event"`
	Data        interface{} `json:"data"`
	Timestamp   time.Time   `json:"timestamp"`
	CollectorID string      `json:"collector_id,omitempty"`
	UserID      uint        `json:"user_id,omitempty"`
}

// PowerDataMessage represents real-time power data
type PowerDataMessage struct {
	CollectorID string    `json:"collector_id"`
	Timestamp   time.Time `json:"timestamp"`
	Voltage     float64   `json:"voltage"`
	Current     float64   `json:"current"`
	Power       float64   `json:"power"`
	Energy      float64   `json:"energy"`
	Frequency   float64   `json:"frequency"`
	PowerFactor float64   `json:"power_factor"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}
	globalHub *Hub
)

// Init initializes the realtime service
func Init(ctx context.Context) {
	globalHub = &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}

	go globalHub.run(ctx)
}

// GetHub returns the global hub instance
func GetHub() *Hub {
	return globalHub
}

// run starts the hub's main loop
func (h *Hub) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

			// Send welcome message
			welcomeMsg := Message{
				Type:      "system",
				Event:     "connected",
				Data:      map[string]string{"status": "connected"},
				Timestamp: time.Now(),
			}

			if data, err := json.Marshal(welcomeMsg); err == nil {
				select {
				case client.send <- data:
				default:
					close(client.send)
					h.mutex.Lock()
					delete(h.clients, client)
					h.mutex.Unlock()
				}
			}

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade connection"})
		return
	}

	// Extract client information from query parameters or headers
	userID := c.GetUint("user_id")
	clientType := c.DefaultQuery("client_type", "user")
	clientID := c.DefaultQuery("client_id", "")

	client := &Client{
		hub:        globalHub,
		conn:       conn,
		send:       make(chan []byte, 256),
		userID:     userID,
		clientID:   clientID,
		clientType: clientType,
	}

	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// HandleSSE handles Server-Sent Events connections
func HandleSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Send initial connection message
	welcomeMsg := Message{
		Type:      "system",
		Event:     "connected",
		Data:      map[string]string{"status": "connected", "transport": "sse"},
		Timestamp: time.Now(),
	}

	if data, err := json.Marshal(welcomeMsg); err == nil {
		c.SSEvent("message", string(data))
		c.Writer.Flush()
	}

	// Keep connection alive and send periodic data
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			// Send heartbeat
			heartbeat := Message{
				Type:      "system",
				Event:     "heartbeat",
				Data:      map[string]interface{}{"timestamp": time.Now()},
				Timestamp: time.Now(),
			}

			if data, err := json.Marshal(heartbeat); err == nil {
				c.SSEvent("heartbeat", string(data))
				c.Writer.Flush()
			}
		}
	}
}

// BroadcastPowerData broadcasts power data to all connected clients
func BroadcastPowerData(data PowerDataMessage) {
	if globalHub == nil {
		return
	}

	message := Message{
		Type:        "data",
		Event:       "power_data",
		Data:        data,
		Timestamp:   time.Now(),
		CollectorID: data.CollectorID,
	}

	if msgData, err := json.Marshal(message); err == nil {
		globalHub.broadcast <- msgData
	}
}

// BroadcastCollectorStatus broadcasts collector status changes
func BroadcastCollectorStatus(collectorID string, isOnline bool) {
	if globalHub == nil {
		return
	}

	message := Message{
		Type:        "status",
		Event:       "collector_status",
		Data:        map[string]interface{}{"collector_id": collectorID, "online": isOnline},
		Timestamp:   time.Now(),
		CollectorID: collectorID,
	}

	if msgData, err := json.Marshal(message); err == nil {
		globalHub.broadcast <- msgData
	}
}

// BroadcastAlert broadcasts alerts to admin users
func BroadcastAlert(alertType, message string, data interface{}) {
	if globalHub == nil {
		return
	}

	alertMsg := Message{
		Type:      "alert",
		Event:     alertType,
		Data:      map[string]interface{}{"message": message, "data": data},
		Timestamp: time.Now(),
	}

	if msgData, err := json.Marshal(alertMsg); err == nil {
		globalHub.broadcast <- msgData
	}
}

// readPump handles reading from the websocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump handles writing to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
