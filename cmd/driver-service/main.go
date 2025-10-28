package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"ride-hail/internal/driver/adapter/handlers"
	"ride-hail/internal/driver/adapter/psql"
	"ride-hail/internal/driver/app/usecase"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/util"
	"syscall"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

func main() {
	log := util.New()

	log.Info("DriverService", "Starting service initialization...")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Config", err)
	}
	log.OK("Config", "Configuration loaded successfully")

	var ch *amqp091.Channel
	// conn, ch, err := mq.ConnectToRMQ(&cfg.RabbitMQ)
	// if err != nil {
	// 	panic(err)
	// }

	// defer conn.Close()
	// defer ch.Close()

	database := db.ConnectToDB(&cfg.Database)
	if database == nil {
		log.Fatal("Database", err)
	}
	defer database.Close()
	log.OK("Database", "Connected successfully")

	repo := psql.NewRepo(database)
	service := usecase.NewService(repo, ch)
	handler := handlers.NewHandler(service)

	mux := handler.Router()

	server := &http.Server{
		Addr:    ":" + "3001",
		Handler: mux,
	}

	go func() {
		log.OK("HTTP", "ride-service running on :3001")
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
