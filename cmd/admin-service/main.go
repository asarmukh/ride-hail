package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ride-hail/internal/admin/api"
	"ride-hail/internal/admin/app"
	"ride-hail/internal/admin/repo"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/util"
)

func main() {
	Run()
}

func Run() {
	log := util.New()

	log.Info("AdminService", "Starting service initialization...")

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

	repository := repo.NewAdminRepo(dbConn)
	service := app.NewAdminService(repository)
	handler := api.NewHandler(service)

	mux := handler.RegisterRoutes()

	server := &http.Server{
		Addr:    ":3004",
		Handler: mux,
	}

	go func() {
		log.OK("HTTP", "admin-service running on :3004")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("AdminService", "Shutting down admin-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("HTTP", err)
	} else {
		log.OK("HTTP", "Server stopped gracefully")
	}

	log.Info("AdminService", "Shutdown complete")
}
