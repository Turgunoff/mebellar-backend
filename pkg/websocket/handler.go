package websocket

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for now (you can restrict this in production)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var jwtSecret string

// SetJWTSecret sets the JWT secret for authentication
func SetJWTSecret(secret string) {
	jwtSecret = secret
}

// HandleWebSocket handles WebSocket connection requests
func HandleWebSocket(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract JWT token from query params or Authorization header
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			log.Printf("❌ WebSocket: No token provided")
			http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
			return
		}

		// 2. Validate JWT token
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			log.Printf("❌ WebSocket: Invalid token: %v", err)
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		// 3. Extract user info from token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			log.Printf("❌ WebSocket: Invalid token claims")
			http.Error(w, "Unauthorized: Invalid token claims", http.StatusUnauthorized)
			return
		}

		userID, _ := claims["user_id"].(string)
		if userID == "" {
			log.Printf("❌ WebSocket: No user_id in token")
			http.Error(w, "Unauthorized: No user_id", http.StatusUnauthorized)
			return
		}

		// 4. Get shop_id from query params
		shopID := r.URL.Query().Get("shop_id")
		if shopID == "" {
			log.Printf("❌ WebSocket: No shop_id provided")
			http.Error(w, "Bad Request: shop_id required", http.StatusBadRequest)
			return
		}

		// 5. Verify user owns this shop
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM seller_profiles WHERE id = $1 AND user_id = $2", shopID, userID).Scan(&count)
		if err != nil || count == 0 {
			log.Printf("❌ WebSocket: User %s doesn't own shop %s", userID, shopID)
			http.Error(w, "Forbidden: Not your shop", http.StatusForbidden)
			return
		}

		// 6. Upgrade to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("❌ WebSocket upgrade error: %v", err)
			return
		}

		// 7. Create client and register with hub
		client := &Client{
			ID:     uuid.New().String(),
			ShopID: shopID,
			UserID: userID,
			Send:   make(chan []byte, 256),
			Hub:    GlobalHub,
		}

		wsConn := NewConnection(conn, client)
		client.Conn = wsConn

		// Register client
		GlobalHub.register <- client

		// Send welcome message
		welcomeMsg := &BroadcastMessage{
			ShopID: shopID,
			Type:   "connected",
			Payload: map[string]interface{}{
				"message":   "WebSocket ulandi! Yangi buyurtmalar real-time keladi.",
				"client_id": client.ID,
				"shop_id":   shopID,
			},
		}

		client.Send <- welcomeMsg.ToJSON()

		log.Printf("✅ WebSocket: Client %s connected for shop %s (user: %s)", client.ID, shopID, userID)

		// Start read/write pumps in goroutines
		go wsConn.WritePump()
		go wsConn.ReadPump()
	}
}

// BroadcastNewOrder broadcasts a new order to all connected clients of a shop
func BroadcastNewOrder(shopID string, payload NewOrderPayload) {
	if GlobalHub != nil {
		GlobalHub.BroadcastToShop(shopID, MessageTypeNewOrder, payload)
	}
}

// BroadcastOrderUpdate broadcasts an order status update
func BroadcastOrderUpdate(shopID string, payload OrderUpdatePayload) {
	if GlobalHub != nil {
		GlobalHub.BroadcastToShop(shopID, MessageTypeOrderUpdate, payload)
	}
}
