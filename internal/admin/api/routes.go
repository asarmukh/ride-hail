package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ride-hail/internal/shared/util"

	jwt "github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("supersecret")

type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (h *Handler) RegisterRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Admin routes with authorization
	mux.Handle("/admin/overview", AdminAuthMiddleware(http.HandlerFunc(h.GetSystemOverview)))
	mux.Handle("/admin/rides/active", AdminAuthMiddleware(http.HandlerFunc(h.GetActiveRides)))

	return mux
}

func AdminAuthMiddleware(next http.Handler) http.Handler {
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

		// Check if user has ADMIN role
		if claims.Role != "ADMIN" {
			util.WriteJSONError(w, "admin access required", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "role", claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
