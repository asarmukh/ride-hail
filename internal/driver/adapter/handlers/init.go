package handlers

import (
	"net/http"

	"ride-hail/internal/driver/app/usecase"
	"ride-hail/internal/shared/middleware"
)

type Handler struct {
	service          usecase.Service
	wsManager        WSManager
	matchingConsumer MatchingConsumer
}

// MatchingConsumer interface for handling driver responses
type MatchingConsumer interface {
	HandleDriverResponse(offerID string, accepted bool, location usecase.LocationResponse) error
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{
		service:   service,
		wsManager: NewWSManager(),
	}
}

func (h *Handler) SetMatchingConsumer(consumer MatchingConsumer) {
	h.matchingConsumer = consumer
}

func (h *Handler) GetWSManager() WSManager {
	return h.wsManager
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /drivers/{user_id}/register", h.RegisterDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/online", h.StartDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/offline", h.FinishDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/location", h.CurrLocationDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/start", h.StartRide)
	mux.HandleFunc("POST /drivers/{driver_id}/complete", h.CompleteRide)
	mux.HandleFunc("GET /ws/drivers/{driver_id}", h.HandleDriverWebSocket)

	// Apply request ID middleware to all routes
	return middleware.RequestID(mux)
}

// RouterWithHealth creates router with health check endpoint
func (h *Handler) RouterWithHealth(healthHandler http.HandlerFunc) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /drivers/{user_id}/register", h.RegisterDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/online", h.StartDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/offline", h.FinishDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/location", h.CurrLocationDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/start", h.StartRide)
	mux.HandleFunc("POST /drivers/{driver_id}/complete", h.CompleteRide)
	mux.HandleFunc("GET /ws/drivers/{driver_id}", h.HandleDriverWebSocket)
	mux.HandleFunc("GET /health", healthHandler)

	// Apply request ID middleware to all routes
	return middleware.RequestID(mux)
}
