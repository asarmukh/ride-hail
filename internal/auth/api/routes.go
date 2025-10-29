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

// RegisterRoutesWithHealth registers routes including health check endpoint
func (h *Handler) RegisterRoutesWithHealth(healthHandler http.HandlerFunc) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", h.Register)
	mux.HandleFunc("/auth/login", h.Login)
	mux.HandleFunc("/health", healthHandler)

	// Apply request ID middleware to all routes
	return middleware.RequestID(mux)
}
