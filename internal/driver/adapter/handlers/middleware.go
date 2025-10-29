package handlers

import (
	"context"
	"net/http"
	"strings"

	"ride-hail/internal/shared/jwt"
	"ride-hail/internal/shared/models"

	jwtLib "github.com/golang-jwt/jwt/v5"
)

type contextKey string

const DriverIDKey contextKey = "driver_id"

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing auth header", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &models.Claims{}

		token, err := jwtLib.ParseWithClaims(tokenString, claims, func(token *jwtLib.Token) (interface{}, error) {
			return jwt.JwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		if claims.Role != "driver" {
			http.Error(w, "invalid role - driver role required", http.StatusForbidden)
			return
		}

		// Extract driver_id from URL and verify it matches token
		driverID := r.PathValue("driver_id")
		if driverID != "" && driverID != claims.Subject {
			http.Error(w, "forbidden - driver_id mismatch", http.StatusForbidden)
			return
		}

		// Add driver_id to context
		ctx := context.WithValue(r.Context(), DriverIDKey, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
