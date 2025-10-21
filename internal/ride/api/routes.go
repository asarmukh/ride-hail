package api

import (
	"net/http"
	"ride-hail/internal/ride/app"
)

type Handler struct {
	service *app.RideService
}

func NewHandler(service *app.RideService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/rides", AuthMiddleware(http.HandlerFunc(h.CreateRideHandler)))
	mux.Handle("/rides/", AuthMiddleware(http.HandlerFunc(h.CancelRideHandler)))
	mux.HandleFunc("/ws/passengers/", h.PassengerWSHandler)
	return mux
}
