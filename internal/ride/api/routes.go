package api

import (
	"net/http"
	"ride-hail/internal/ride/app"
	"ride-hail/internal/ride/repo"
)

type Handler struct {
	service *app.RideService
}

func NewHandler(service *app.RideService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rideRepo *repo.RideRepo) *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/rides", AuthMiddleware(rideRepo)(http.HandlerFunc(h.CreateRideHandler)))
	mux.Handle("/rides/", AuthMiddleware(rideRepo)(http.HandlerFunc(h.CancelRideHandler)))
	mux.HandleFunc("/ws/passengers/", h.PassengerWSHandler)
	return mux
}
