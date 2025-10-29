package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-hail/internal/auth/api"
	"ride-hail/internal/auth/app"
	"ride-hail/internal/auth/repo"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/util"
)

func main() {
	Run()
}

func Run() {
	log := util.NewLogger("auth-service")

	log.Info("service_start", "Starting service initialization")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("config_load", "Failed to load configuration", err)
	}
	log.OK("config_load", "Configuration loaded successfully")

	dbConn := db.ConnectToDB(&cfg.Database)
	if dbConn == nil {
		log.Fatal("database_connect", "Failed to connect to database", err)
	}
	defer dbConn.Close()
	log.OK("database_connect", "Connected successfully")

	repository := repo.NewAuthRepo(dbConn)
	service := app.NewAuthService(repository, log)
	handler := api.NewHandler(service)

	mux := handler.RegisterRoutes()

	server := &http.Server{
		Addr:    ":4000",
		Handler: mux,
	}

	go func() {
		log.OK("http_start", "Auth service running on :4000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http_error", "HTTP server error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("service_shutdown", "Shutting down auth-service")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("service_shutdown", "Error during shutdown", err)
	} else {
		log.OK("service_shutdown", "Server stopped gracefully")
	}

	log.Info("service_shutdown", "Shutdown complete")
}
