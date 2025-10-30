package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-hail/internal/driver/adapter/handlers"
	"ride-hail/internal/driver/adapter/psql"
	"ride-hail/internal/driver/app/usecase"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/health"
	"ride-hail/internal/shared/mq"
	"ride-hail/internal/shared/util"

	driverRmq "ride-hail/internal/driver/adapter/rmq"

	"github.com/rabbitmq/amqp091-go"
)

func main() {
	Run()
}

func Run() {
	log := util.NewLogger("driver-service")

	log.Info("service_start", "Starting service initialization")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("config_load", "Failed to load configuration", err)
	}
	log.OK("config_load", "Configuration loaded successfully")

	var ch *amqp091.Channel
	conn, ch, err := mq.ConnectToRMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatal("rabbitmq_connect", "Failed to connect to RabbitMQ", err)
	}
	defer conn.Close()
	defer ch.Close()
	log.OK("rabbitmq_connect", "Connected successfully")

	database := db.ConnectToDB(&cfg.Database)
	if database == nil {
		log.Fatal("database_connect", "Failed to connect to database", err)
	}
	defer database.Close()
	log.OK("database_connect", "Connected successfully")

	repo := psql.NewRepo(database)
	broker := driverRmq.NewBroker(ch)
	service := usecase.NewService(repo, broker)
	handler := handlers.NewHandler(service)
	wsManager := handler.GetWSManager()

	match := usecase.NewMatchingConsumer(service, repo, ch, wsManager)

	// Connect the matching consumer to the handler so it can process driver responses
	handler.SetMatchingConsumer(match)

	go match.Start()

	healthHandler := health.Handler("driver-service", database, conn)
	mux := handler.Router(healthHandler)

	server := &http.Server{
		Addr:    ":" + "3001",
		Handler: mux,
	}

	go func() {
		log.OK("http_start", "Driver service running on :3001")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http_error", "HTTP server error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("service_shutdown", "Shutting down driver-service")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("service_shutdown", "Error during shutdown", err)
	} else {
		log.OK("service_shutdown", "Server stopped gracefully")
	}
	log.Info("service_shutdown", "Shutdown complete")
}
