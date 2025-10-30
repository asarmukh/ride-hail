package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"ride-hail/internal/driver/app/usecase"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var jwtSecret = []byte("supersecret")

type WSManager interface {
	SendOfferToDriver(driverID string, offer interface{}) error
	SendRideDetailsToDriver(driverID string, details RideDetails) error
	SendMessageToDriver(driverID string, message interface{}) error
	AddConn(conn *websocket.Conn, driverID string)
	DeleteConn(driverID string)
}

type WebSocketManager struct {
	drivers map[string]*websocket.Conn
	mu      sync.RWMutex
}

func NewWSManager() WSManager {
	return &WebSocketManager{
		drivers: make(map[string]*websocket.Conn),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type AuthMessage struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type WSResponse struct {
	Type    string      `json:"type"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type RideResponse struct {
	Type            string   `json:"type"`
	OfferID         string   `json:"offer_id"`
	RideID          string   `json:"ride_id"`
	Accepted        bool     `json:"accepted"`
	CurrentLocation Location `json:"current_location"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type LocationUpdate struct {
	Type           string  `json:"type"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters"`
	SpeedKmh       float64 `json:"speed_kmh,omitempty"`
	HeadingDegrees float64 `json:"heading_degrees,omitempty"`
}

type RideOffer struct {
	Type                string           `json:"type"`
	OfferID             string           `json:"offer_id"`
	RideID              string           `json:"ride_id"`
	RideNumber          string           `json:"ride_number"`
	PassengerName       string           `json:"passenger_name"`
	PickupLocation      LocationWithAddr `json:"pickup_location"`
	DestinationLocation LocationWithAddr `json:"destination_location"`
	EstimatedFare       float64          `json:"estimated_fare"`
	EstimatedDistance   float64          `json:"estimated_distance_km"`
	ExpiresAt           string           `json:"expires_at"`
}

type LocationWithAddr struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

type RideDetails struct {
	Type                string           `json:"type"`
	RideID              string           `json:"ride_id"`
	PassengerInfo       PassengerInfo    `json:"passenger_info"`
	PickupLocation      LocationWithAddr `json:"pickup_location"`
	DestinationLocation LocationWithAddr `json:"destination_location"`
	EstimatedFare       float64          `json:"estimated_fare"`
}

type PassengerInfo struct {
	Name   string  `json:"name"`
	Phone  string  `json:"phone"`
	Rating float64 `json:"rating,omitempty"`
}

type Claims struct {
	DriverID string `json:"sub"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// HandleDriverWebSocket handles the WebSocket connection for a driver
func (h *Handler) HandleDriverWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract driver_id from URL path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[1] != "drivers" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	driverID := parts[2]

	// // Validate driverID is a valid UUID
	// if _, err := uuid.Parse(driverID); err != nil {
	// 	http.Error(w, "invalid driver_id", http.StatusBadRequest)
	// 	return
	// }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("New WebSocket connection from driver: %s", driverID)

	// Authenticate with timeout
	if !h.authenticateDriverWithTimeout(conn, driverID) {
		log.Printf("Driver %s authentication failed or timed out", driverID)
		return
	}

	// Register driver connection
	h.registerDriver(driverID, conn)
	defer h.unregisterDriver(driverID)

	log.Printf("Driver %s successfully authenticated and connected", driverID)

	// Start ping/pong mechanism
	stopPing := make(chan bool)
	go h.startPingPong(conn, stopPing)
	defer func() { stopPing <- true }()

	// Handle incoming messages
	h.handleDriverMessages(conn, driverID)
}

// authenticateDriverWithTimeout authenticates the driver within 5 seconds
func (h *Handler) authenticateDriverWithTimeout(conn *websocket.Conn, driverID string) bool {
	authenticated := false
	authTimer := time.NewTimer(5 * time.Second)
	authChan := make(chan string, 1)

	// Read authentication message
	go func() {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading auth message: %v", err)
			return
		}

		var authMsg AuthMessage
		if err := json.Unmarshal(msg, &authMsg); err != nil {
			log.Printf("Error unmarshaling auth message: %v", err)
			return
		}

		if authMsg.Type == "auth" {
			authChan <- authMsg.Token
		}
	}()

	// Wait for auth or timeout
	select {
	case tokenStr := <-authChan:
		if h.validateDriverToken(tokenStr, driverID) {
			authenticated = true
			_ = conn.WriteJSON(WSResponse{Type: "auth_success", Message: "authenticated"})
		} else {
			_ = conn.WriteJSON(WSResponse{Type: "error", Message: "invalid token or unauthorized"})
		}
	case <-authTimer.C:
		_ = conn.WriteJSON(WSResponse{Type: "error", Message: "authentication timeout"})
	}

	return authenticated
}

// validateDriverToken validates the JWT token and checks if it matches the driver_id
func (h *Handler) validateDriverToken(headerToken, driverID string) bool {
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
		log.Printf("Token validation failed: %v", err)
		return false
	}

	// Verify the token is for a driver and matches the requested driver_id
	if claims.Role != "DRIVER" {
		log.Printf("Token role is not DRIVER: %s", claims.Role)
		return false
	}

	if claims.DriverID != driverID {
		log.Printf("Token driver_id (%s) does not match requested driver_id (%s)", claims.DriverID, driverID)
		return false
	}

	return true
}

// registerDriver adds driver connection to the manager
func (h *Handler) registerDriver(driverID string, conn *websocket.Conn) {
	if h.wsManager == nil {
		h.wsManager = NewWSManager()
	}
	h.wsManager.AddConn(conn, driverID)
}

// unregisterDriver removes driver connection from the manager
func (h *Handler) unregisterDriver(driverID string) {
	if h.wsManager == nil {
		return
	}
	h.wsManager.DeleteConn(driverID)
	log.Printf("Driver %s disconnected", driverID)
}

// startPingPong sends ping messages every 30 seconds
func (h *Handler) startPingPong(conn *websocket.Conn, stop chan bool) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Set pong handler to extend read deadline
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Set initial read deadline
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("Ping failed: %v", err)
				return
			}
		case <-stop:
			return
		}
	}
}

// handleDriverMessages processes incoming messages from drivers
func (h *Handler) handleDriverMessages(conn *websocket.Conn, driverID string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse message type
		var baseMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg, &baseMsg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		switch baseMsg.Type {
		case "ride_response":
			h.handleRideResponse(msg, driverID)
		case "location_update":
			h.handleLocationUpdate(msg, driverID)
		default:
			log.Printf("Unknown message type: %s", baseMsg.Type)
		}
	}
}

// handleRideResponse processes driver's response to ride offer
func (h *Handler) handleRideResponse(msg []byte, driverID string) {
	var response RideResponse
	if err := json.Unmarshal(msg, &response); err != nil {
		log.Printf("Error unmarshaling ride response: %v", err)
		return
	}

	log.Printf("Received ride response from driver %s: ride_id=%s, accepted=%v",
		driverID, response.RideID, response.Accepted)

	// Send response to matching consumer
	if h.matchingConsumer != nil {
		location := usecase.LocationResponse{
			Latitude:  response.CurrentLocation.Latitude,
			Longitude: response.CurrentLocation.Longitude,
		}
		if err := h.matchingConsumer.HandleDriverResponse(response.OfferID, response.Accepted, location); err != nil {
			log.Printf("Error handling driver response: %v", err)
		}
	} else {
		log.Printf("Warning: matching consumer not set, cannot process driver response")
	}
}

// handleLocationUpdate processes driver's location update
func (h *Handler) handleLocationUpdate(msg []byte, driverID string) {
	// Check rate limiting
	if !h.service.CanUpdateLocation(driverID) {
		log.Printf("Rate limit exceeded for driver %s", driverID)
		// Send rate limit error to driver
		if h.wsManager != nil {
			_ = h.wsManager.SendMessageToDriver(driverID, WSResponse{
				Type:    "error",
				Message: "Rate limit exceeded. Max 1 update per 3 seconds",
			})
		}
		return
	}

	var update LocationUpdate
	if err := json.Unmarshal(msg, &update); err != nil {
		log.Printf("Error unmarshaling location update: %v", err)
		return
	}

	log.Printf("Received location update from driver %s: lat=%.6f, lng=%.6f",
		driverID, update.Latitude, update.Longitude)

	// Store location in database and publish to RabbitMQ
	ctx := context.Background()
	// driverUUID, err := uuid.Parse(driverID)
	// if err != nil {
	// 	log.Printf("Invalid driver UUID: %v", err)
	// 	return
	// }

	if err := h.service.UpdateDriverLocation(ctx, driverID, update.Latitude, update.Longitude,
		update.AccuracyMeters, update.SpeedKmh, update.HeadingDegrees); err != nil {
		log.Printf("Error updating driver location: %v", err)
	}

	// Location is now published to location_fanout in UpdateDriverLocation method
}

// SendOfferToDriver sends a ride offer to a specific driver via WebSocket
func (m *WebSocketManager) SendOfferToDriver(driverID string, offer interface{}) error {
	m.mu.RLock()
	conn, ok := m.drivers[driverID]
	m.mu.RUnlock()

	if !ok {
		return nil // Driver not connected, skip
	}

	return conn.WriteJSON(offer)
}

// SendRideDetailsToDriver sends ride details after driver accepts
func (m *WebSocketManager) SendRideDetailsToDriver(driverID string, details RideDetails) error {
	m.mu.RLock()
	conn, ok := m.drivers[driverID]
	m.mu.RUnlock()

	if !ok {
		return nil // Driver not connected
	}

	return conn.WriteJSON(details)
}

// SendMessageToDriver sends any message to a driver
func (m *WebSocketManager) SendMessageToDriver(driverID string, message interface{}) error {
	m.mu.RLock()
	conn, ok := m.drivers[driverID]
	m.mu.RUnlock()

	if !ok {
		return nil // Driver not connected
	}

	return conn.WriteJSON(message)
}

func (m *WebSocketManager) AddConn(conn *websocket.Conn, driverID string) {
	m.mu.Lock()
	m.drivers[driverID] = conn
	m.mu.Unlock()
}

func (m *WebSocketManager) DeleteConn(driverID string) {
	m.mu.Lock()
	delete(m.drivers, driverID)
	m.mu.Unlock()
}
