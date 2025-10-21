package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var activeConnections = make(map[string]*websocket.Conn)

type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type WSResponse struct {
	Type    string      `json:"type"`
	Message string      `json:"message"`
	Payload interface{} `json:"payload,omitempty"`
}

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
			_ = conn.WriteJSON(WSResponse{Type: "auth_success", Message: "authenticated"})
		} else {
			_ = conn.WriteJSON(WSResponse{Type: "error", Message: "invalid token"})
			return
		}
	case <-tokenTimer.C:
		_ = conn.WriteJSON(WSResponse{Type: "error", Message: "auth timeout"})
		return
	}

	if !authenticated {
		return
	}

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
