package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"ride-hail/internal/auth/api"
	"ride-hail/internal/auth/app"
	"ride-hail/internal/auth/repo"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/util"
	"syscall"
	"time"
)

func main() {
	log := util.New()

	log.Info("AuthService", "Starting service initialization...")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Config", err)
	}
	log.OK("Config", "Configuration loaded successfully")

	dbConn := db.ConnectToDB(&cfg.Database)
	if dbConn == nil {
		log.Fatal("Database", err)
	}
	defer dbConn.Close()
	log.OK("Database", "Connected successfully")

	repository := repo.NewAuthRepo(dbConn)
	service := app.NewAuthService(repository, log)
	handler := api.NewHandler(service)

	mux := handler.RegisterRoutes()

	server := &http.Server{
		Addr:    ":4000",
		Handler: mux,
	}

	go func() {
		log.OK("HTTP", "auth-service running on :4000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("AuthService", "Shutting down auth-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("HTTP", err)
	} else {
		log.OK("HTTP", "Server stopped gracefully")
	}

	log.Info("AuthService", "Shutdown complete")
}
