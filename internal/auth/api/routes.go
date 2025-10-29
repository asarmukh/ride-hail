package api

import (
	"net/http"

	"ride-hail/internal/auth/app"
	"ride-hail/internal/shared/middleware"
)

type Handler struct {
	service *app.AuthService
}

func NewHandler(s *app.AuthService) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", h.Register)
	mux.HandleFunc("/auth/login", h.Login)

	// Apply request ID middleware to all routes
	return middleware.RequestID(mux)
}
