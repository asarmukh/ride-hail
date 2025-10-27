package handlers

import (
	"net/http"
)

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// authHeader := r.Header.Get("Authorization")
		// if authHeader == "" {
		// 	http.Error(w, "missing auth header", http.StatusUnauthorized)
		// 	return
		// }

		// tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		// claims := &models.Claims{}

		// // token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// // 	return jwt2.JwtKey, nil
		// // })

		// if err != nil || !token.Valid {
		// 	http.Error(w, "invalid token", http.StatusUnauthorized)
		// 	return
		// }

		// if claims.Role != "driver" {
		// 	http.Error(w, "invalid role", http.StatusUnauthorized)
		// 	return
		// }

		next.ServeHTTP(w, r)
	})
}
