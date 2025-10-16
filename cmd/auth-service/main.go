package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"ride-hail/internal/auth/api"
	"ride-hail/internal/auth/app"
	"ride-hail/internal/auth/repo"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	dbConn := db.ConnectToDB(&cfg.Database)
	defer dbConn.Close()

	repository := repo.NewAuthRepo(dbConn)
	service := app.NewAuthService(repository)
	handler := api.NewHandler(service)

	mux := handler.RegisterRoutes()

	server := &http.Server{
		Addr:    ":4000",
		Handler: mux,
	}

	go func() {
		log.Println("auth-service running on :4000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("auth-service failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("auth-service stopped gracefully")
}
