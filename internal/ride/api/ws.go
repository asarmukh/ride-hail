package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSManager struct {
	passengers map[string]*websocket.Conn
	mu         sync.RWMutex
}

func NewWSManager() *WSManager {
	return &WSManager{
		passengers: make(map[string]*websocket.Conn),
	}
}

var (
	activeConnections = make(map[string]*websocket.Conn)
	globalWSManager   = NewWSManager()
)

func (h *Handler) PassengerWSHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[1] != "passengers" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	passengerID := parts[2]

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("New WS connection from passenger: %s", passengerID)

	authenticated := false
	tokenTimer := time.NewTimer(5 * time.Second)

	authChan := make(chan string)

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("read error: %v", err)
				break
			}

			var authMsg AuthMessage
			if err := json.Unmarshal(msg, &authMsg); err != nil {
				continue
			}

			if authMsg.Type == "auth" {
				authChan <- authMsg.Token
				return
			}
		}
	}()

	select {
	case tokenStr := <-authChan:
		if validateWebSocketToken(tokenStr, passengerID) {
			authenticated = true
			activeConnections[passengerID] = conn
			globalWSManager.RegisterPassenger(passengerID, conn)
			_ = conn.WriteJSON(WSResponse{Type: "auth_success", Message: "authenticated"})
		} else {
			_ = conn.WriteJSON(WSResponse{Type: "error", Message: "invalid token or passanger_id"})
			return
		}
	case <-tokenTimer.C:
		_ = conn.WriteJSON(WSResponse{Type: "error", Message: "auth timeout"})
		return
	}

	if !authenticated {
		return
	}

	// Set read deadline and pong handler
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		// Reset deadline when pong received
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start a goroutine to read messages (even if we only care about pongs)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("read error: %v", err)
				delete(activeConnections, passengerID)
				break
			}
		}
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("ping failed: %v", err)
				delete(activeConnections, passengerID)
				return
			}
		}
	}
}

func validateWebSocketToken(headerToken, passengerID string) bool {
	parts := strings.Split(headerToken, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false
	}

	tokenStr := parts[1]
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return false
	}

	return claims.PassengerID == passengerID
}

func SendToPassenger(ctx context.Context, passengerID string, event WSResponse) error {
	conn, ok := activeConnections[passengerID]
	if !ok {
		return nil
	}
	return conn.WriteJSON(event)
}

func (m *WSManager) SendToPassenger(passengerID string, message interface{}) error {
	m.mu.RLock()
	conn, ok := m.passengers[passengerID]
	m.mu.RUnlock()

	if !ok {
		// Passenger not connected - this is OK, just log at debug level
		log.Printf("Passenger %s not connected", passengerID)
		return nil
	}

	if err := conn.WriteJSON(message); err != nil {
		// Connection dead, remove from map
		m.mu.Lock()
		delete(m.passengers, passengerID)
		m.mu.Unlock()
		return err
	}

	return nil
}

func (m *WSManager) RegisterPassenger(passengerID string, conn *websocket.Conn) {
	m.mu.Lock()
	m.passengers[passengerID] = conn
	m.mu.Unlock()
}

func (m *WSManager) UnregisterPassenger(passengerID string) {
	m.mu.Lock()
	delete(m.passengers, passengerID)
	m.mu.Unlock()
}

func GetGlobalWSManager() *WSManager {
	return globalWSManager
}
