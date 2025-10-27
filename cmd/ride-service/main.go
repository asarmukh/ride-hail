package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"ride-hail/internal/ride/api"
	"ride-hail/internal/ride/app"
	"ride-hail/internal/ride/consumer"
	"ride-hail/internal/ride/repo"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/mq"
	"ride-hail/internal/shared/util"
	"syscall"
	"time"
)

func main() {
	log := util.New()

	log.Info("RideService", "Starting service initialization...")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Config", err)
	}
	log.OK("Config", "Configuration loaded successfully")

	db := db.ConnectToDB(&cfg.Database)
	if db == nil {
		log.Fatal("Database", err)
	}
	defer db.Close()
	log.OK("Database", "Connected successfully")

	rmqConn, rmqCh, err := mq.ConnectToRMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatal("RabbitMQ", err)
	}
	defer rmqConn.Close()
	defer rmqCh.Close()
	log.OK("RabbitMQ", "Connected successfully")

	publisher := mq.NewPublisher(rmqCh)
	repository := repo.NewRideRepo(db)

	service := app.NewRideService(repository, publisher, log)
	handler := api.NewHandler(service)

	consumer := consumer.NewDriverResponseConsumer(service, rmqCh)
	if err := consumer.Start(context.Background()); err != nil {
		log.Fatal("DriverResponseConsumer", err)
	}

	log.OK("DriverResponseConsumer", "Started successfully")

	mux := handler.RegisterRoutes(repository)

	server := &http.Server{
		Addr:    ":" + "3000",
		Handler: mux,
	}

	go func() {
		log.OK("HTTP", "ride-service running on :3000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("RideService", "Shutting down ride-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("HTTP", err)
	} else {
		log.OK("HTTP", "Server stopped gracefully")
	}
	log.Info("RideService", "Shutdown complete")
}
