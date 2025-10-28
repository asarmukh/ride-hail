package api

import (
	"net/http"
	"ride-hail/internal/auth/app"
)

type Handler struct {
	service *app.AuthService
}

func NewHandler(s *app.AuthService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", h.Register)
	mux.HandleFunc("/auth/login", h.Login)
	return mux
}
