package api

import (
	"net/http"

	"ride-hail/internal/ride/app"
	"ride-hail/internal/ride/repo"
	"ride-hail/internal/shared/middleware"
)

type Handler struct {
	service *app.RideService
	
}

func NewHandler(service *app.RideService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rideRepo *repo.RideRepo) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/rides", AuthMiddleware(rideRepo)(http.HandlerFunc(h.CreateRideHandler)))
	mux.Handle("/rides/", AuthMiddleware(rideRepo)(http.HandlerFunc(h.CancelRideHandler)))
	mux.HandleFunc("/ws/passengers/", h.PassengerWSHandler)

	// Apply request ID middleware to all routes
	return middleware.RequestID(mux)
}

// RegisterRoutesWithHealth registers routes including health check endpoint
func (h *Handler) RegisterRoutesWithHealth(rideRepo *repo.RideRepo, healthHandler http.HandlerFunc) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/rides", AuthMiddleware(rideRepo)(http.HandlerFunc(h.CreateRideHandler)))
	mux.Handle("/rides/", AuthMiddleware(rideRepo)(http.HandlerFunc(h.CancelRideHandler)))
	mux.HandleFunc("/ws/passengers/", h.PassengerWSHandler)
	mux.HandleFunc("/health", healthHandler)

	// Apply request ID middleware to all routes
	return middleware.RequestID(mux)
}
