package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabbitmq/amqp091-go"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// Handler creates a health check handler for a service
func Handler(serviceName string, db *pgxpool.Pool, rmqConn *amqp091.Connection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health := HealthResponse{
			Status:    "healthy",
			Service:   serviceName,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Checks:    make(map[string]string),
		}

		// Check database
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()

			if err := db.Ping(ctx); err != nil {
				health.Status = "unhealthy"
				health.Checks["database"] = "down"
			} else {
				health.Checks["database"] = "up"
			}
		}

		// Check RabbitMQ
		if rmqConn != nil {
			if rmqConn.IsClosed() {
				health.Status = "unhealthy"
				health.Checks["rabbitmq"] = "down"
			} else {
				health.Checks["rabbitmq"] = "up"
			}
		}

		// Set HTTP status code
		statusCode := http.StatusOK
		if health.Status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(health)
	}
}

// HandlerWithoutRabbitMQ creates a health check handler without RabbitMQ check
func HandlerWithoutRabbitMQ(serviceName string, db *pgxpool.Pool) http.HandlerFunc {
	return Handler(serviceName, db, nil)
}
