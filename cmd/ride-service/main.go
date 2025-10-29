package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-hail/internal/ride/api"
	"ride-hail/internal/ride/app"
	"ride-hail/internal/ride/consumer"
	"ride-hail/internal/ride/repo"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/mq"
	"ride-hail/internal/shared/util"
)

func main() {
	Run()
}

func Run() {
	log := util.NewLogger("ride-service")

	log.Info("service_start", "Starting service initialization")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("config_load", "Failed to load configuration", err)
	}
	log.OK("config_load", "Configuration loaded successfully")

	db := db.ConnectToDB(&cfg.Database)
	if db == nil {
		log.Fatal("database_connect", "Failed to connect to database", err)
	}
	defer db.Close()
	log.OK("database_connect", "Connected successfully")

	rmqConn, rmqCh, err := mq.ConnectToRMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatal("rabbitmq_connect", "Failed to connect to RabbitMQ", err)
	}
	defer rmqConn.Close()
	defer rmqCh.Close()
	log.OK("rabbitmq_connect", "Connected successfully")

	publisher := mq.NewPublisher(rmqCh)
	repository := repo.NewRideRepo(db)

	service := app.NewRideService(repository, publisher, log)
	handler := api.NewHandler(service)

	// Get WebSocket manager for location updates
	wsManager := api.GetGlobalWSManager()

	// Start driver response consumer
	driverResponseConsumer := consumer.NewDriverResponseConsumer(service, rmqCh, wsManager)
	if err := driverResponseConsumer.Start(context.Background()); err != nil {
		log.Fatal("consumer_start", "Failed to start driver response consumer", err)
	}
	log.OK("consumer_start", "Driver response consumer started successfully")

	// Start location updates consumer
	locationConsumer := consumer.NewLocationConsumer(service, rmqCh, wsManager)
	if err := locationConsumer.Start(context.Background()); err != nil {
		log.Fatal("consumer_start", "Failed to start location consumer", err)
	}
	log.OK("consumer_start", "Location consumer started successfully")

	// Start ride status updates consumer
	rideStatusConsumer := consumer.NewRideStatusConsumer(service, rmqCh, wsManager)
	if err := rideStatusConsumer.Start(context.Background()); err != nil {
		log.Fatal("consumer_start", "Failed to start ride status consumer", err)
	}
	log.OK("consumer_start", "Ride status consumer started successfully")

	mux := handler.RegisterRoutes(repository)

	server := &http.Server{
		Addr:    ":" + "3000",
		Handler: mux,
	}

	go func() {
		log.OK("http_start", "Ride service running on :3000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http_error", "HTTP server error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("service_shutdown", "Shutting down ride-service")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("service_shutdown", "Error during shutdown", err)
	} else {
		log.OK("service_shutdown", "Server stopped gracefully")
	}
	log.Info("service_shutdown", "Shutdown complete")
}
