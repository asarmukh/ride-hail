package handlers

import (
	"net/http"

	"ride-hail/internal/driver/app/usecase"
)

type Handler struct {
	service usecase.Service
}

func NewHandler(service usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Router() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /drivers/register/{user_id}", h.RegisterDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/online", h.StartDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/offline", h.FinishDriver)
	mux.HandleFunc("POST /drivers/{driver_id}/location", h.CurrLocationDriver)

	return mux
}
