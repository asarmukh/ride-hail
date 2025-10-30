package middleware

import (
	"context"
	"net/http"
	"ride-hail/internal/shared/util"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestID middleware generates or propagates X-Request-ID headers
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing correlation ID in header
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID, _ = util.GenerateUUID()
		}

		// Add to context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Add to response header
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from the context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
