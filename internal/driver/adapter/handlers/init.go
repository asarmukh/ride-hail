package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ride-hail/internal/driver/adapter/psql"
	"ride-hail/internal/driver/app/usecase"
	"ride-hail/internal/shared/middleware"
	"ride-hail/internal/shared/util"

	jwt "github.com/golang-jwt/jwt/v5"
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

// RouterWithHealth creates router with health check endpoint
func (h *Handler) Router(rideRepo *psql.DriveRepo) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("POST /drivers/{driver_id}/register", AuthMiddleware(rideRepo)(http.HandlerFunc(h.RegisterDriver)))
	mux.Handle("POST /drivers/{driver_id}/online", AuthMiddleware(rideRepo)(http.HandlerFunc(h.StartDriver)))
	mux.Handle("POST /drivers/{driver_id}/offline", AuthMiddleware(rideRepo)(http.HandlerFunc(h.FinishDriver)))
	mux.Handle("POST /drivers/{driver_id}/location", AuthMiddleware(rideRepo)(http.HandlerFunc(h.CurrLocationDriver)))
	mux.Handle("POST /drivers/{driver_id}/start", AuthMiddleware(rideRepo)(http.HandlerFunc(h.StartRide)))
	mux.Handle("POST /drivers/{driver_id}/complete", AuthMiddleware(rideRepo)(http.HandlerFunc(h.CompleteRide)))
	mux.Handle("GET /ws/drivers/{driver_id}", AuthMiddleware(rideRepo)(http.HandlerFunc(h.HandleDriverWebSocket)))

	// Apply request ID middleware to all routes
	return middleware.RequestID(mux)
}

func AuthMiddleware(rideRepo *psql.DriveRepo) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				util.WriteJSONError(w, "missing Authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				util.WriteJSONError(w, "invalid Authorization format", http.StatusUnauthorized)
				return
			}

			tokenStr := parts[1]
			claims := &Claims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})
			if err != nil || !token.Valid {
				util.WriteJSONError(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			if time.Now().After(claims.ExpiresAt.Time) {
				util.WriteJSONError(w, "token expired", http.StatusUnauthorized)
				return
			}

			exists, err := rideRepo.Exists(r.Context(), claims.DriverID)
			if err != nil {
				util.WriteJSONError(w, "failed to check user existence", http.StatusInternalServerError)
				return
			}
			if !exists {
				util.WriteJSONError(w, "user not found or deleted", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), "passenger_id", claims.DriverID)
			ctx = context.WithValue(ctx, "role", claims.Role)
			ctx = context.WithValue(ctx, "token_exp", claims.ExpiresAt.Time)

			if time.Now().After(claims.ExpiresAt.Time) {
				util.WriteJSONError(w, "token expired", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
