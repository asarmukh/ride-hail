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

	mux.HandleFunc("/rides", h.CreateRideHandler)
	mux.HandleFunc("/rides/", h.CancelRideHandler)
	return mux
}
