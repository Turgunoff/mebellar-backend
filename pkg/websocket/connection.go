package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Connection wraps the WebSocket connection
type Connection struct {
	ws     *websocket.Conn
	client *Client
}

// NewConnection creates a new Connection
func NewConnection(ws *websocket.Conn, client *Client) *Connection {
	return &Connection{
		ws:     ws,
		client: client,
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Connection) ReadPump() {
	defer func() {
		c.client.Hub.unregister <- c.client
		c.ws.Close()
	}()

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Handle incoming messages if needed
		log.Printf("📨 WebSocket message from client %s: %s", c.client.ID, string(message))
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Connection) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.client.Send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.ws.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ToJSON converts BroadcastMessage to JSON bytes
func (m *BroadcastMessage) ToJSON() []byte {
	data, err := json.Marshal(m)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return []byte("{}")
	}
	return data
}

// WebSocket Message Types
const (
	MessageTypeNewOrder     = "new_order"
	MessageTypeOrderUpdate  = "order_update"
	MessageTypeOrderDeleted = "order_deleted"
)

// NewOrderPayload represents the payload for new order notifications
type NewOrderPayload struct {
	OrderID      string  `json:"order_id"`
	ClientName   string  `json:"client_name"`
	ClientPhone  string  `json:"client_phone"`
	TotalAmount  float64 `json:"total_amount"`
	ProductCount int     `json:"product_count"`
	ProductName  string  `json:"product_name"`
	ProductImage string  `json:"product_image"`
	CreatedAt    string  `json:"created_at"`
}

// OrderUpdatePayload represents the payload for order status updates
type OrderUpdatePayload struct {
	OrderID   string `json:"order_id"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}
