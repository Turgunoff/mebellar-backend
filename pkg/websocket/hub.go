package websocket

import (
	"log"
	"sync"
)

// Client represents a WebSocket client connection
type Client struct {
	ID     string
	ShopID string
	UserID string
	Send   chan []byte
	Hub    *Hub
	Conn   *Connection
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	// Registered clients by shop_id
	clients map[string]map[*Client]bool

	// Broadcast channel
	broadcast chan *BroadcastMessage

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// BroadcastMessage represents a message to broadcast to a shop
type BroadcastMessage struct {
	ShopID  string
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Global Hub instance
var GlobalHub *Hub

// NewHub creates a new Hub instance
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan *BroadcastMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the Hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.ShopID] == nil {
				h.clients[client.ShopID] = make(map[*Client]bool)
			}
			h.clients[client.ShopID][client] = true
			h.mu.Unlock()
			log.Printf("🔌 WebSocket: Client registered for shop %s (Total: %d)", client.ShopID, len(h.clients[client.ShopID]))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ShopID]; ok {
				if _, ok := h.clients[client.ShopID][client]; ok {
					delete(h.clients[client.ShopID], client)
					close(client.Send)
					if len(h.clients[client.ShopID]) == 0 {
						delete(h.clients, client.ShopID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("🔌 WebSocket: Client unregistered from shop %s", client.ShopID)

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[message.ShopID]
			h.mu.RUnlock()

			if clients != nil {
				for client := range clients {
					select {
					case client.Send <- message.ToJSON():
					default:
						h.mu.Lock()
						close(client.Send)
						delete(h.clients[message.ShopID], client)
						h.mu.Unlock()
					}
				}
				log.Printf("📡 WebSocket: Broadcasted '%s' to %d clients in shop %s", message.Type, len(clients), message.ShopID)
			}
		}
	}
}

// BroadcastToShop sends a message to all clients of a specific shop
func (h *Hub) BroadcastToShop(shopID string, messageType string, payload interface{}) {
	msg := &BroadcastMessage{
		ShopID:  shopID,
		Type:    messageType,
		Payload: payload,
	}
	h.broadcast <- msg
}

// GetClientCount returns the number of connected clients for a shop
func (h *Hub) GetClientCount(shopID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[shopID])
}

// InitGlobalHub initializes the global hub instance
func InitGlobalHub() {
	GlobalHub = NewHub()
	go GlobalHub.Run()
	log.Println("✅ WebSocket Hub initialized!")
}
