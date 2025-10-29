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
	mux.HandleFunc("/auth/passenger/login", h.UserLogin)
	mux.HandleFunc("/auth/driver/login", h.LoginDriver)
	return mux
}
