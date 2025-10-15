package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"ride-hail/internal/ride/api"
	"ride-hail/internal/ride/app"
	"ride-hail/internal/ride/repo"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/mq"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db := db.ConnectToDB(&cfg.Database)
	defer db.Close()

	rmqConn, rmqCh, err := mq.ConnectToRMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatalf("RabbitMQ connection failed: %v", err)
	}
	defer rmqConn.Close()
	defer rmqCh.Close()

	if err := rmqCh.ExchangeDeclare(
		"ride_topic",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		log.Fatalf("failed to declare exchange: %v", err)
	}

	publisher := mq.NewPublisher(rmqCh)
	repository := repo.NewRideRepo(db)

	service := app.NewRideService(repository, publisher)
	handler := api.NewHandler(service)

	mux := handler.RegisterRoutes()

	server := &http.Server{
		Addr:    ":" + "3000",
		Handler: mux,
	}

	go func() {
		log.Printf("ride-service running on :%s", "3000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down ride-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Println("ride-service stopped gracefully")
}
