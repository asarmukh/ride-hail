package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	adminApi "ride-hail/internal/admin/api"
	adminApp "ride-hail/internal/admin/app"
	adminRepo "ride-hail/internal/admin/repo"
	"ride-hail/internal/auth/api"
	authApp "ride-hail/internal/auth/app"
	authRepo "ride-hail/internal/auth/repo"
	"ride-hail/internal/driver/adapter/handlers"
	"ride-hail/internal/driver/adapter/psql"
	driverRmq "ride-hail/internal/driver/adapter/rmq"
	"ride-hail/internal/driver/app/usecase"
	rideApi "ride-hail/internal/ride/api"
	rideApp "ride-hail/internal/ride/app"
	"ride-hail/internal/ride/consumer"
	rideRepo "ride-hail/internal/ride/repo"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/mq"
	"ride-hail/internal/shared/util"
)

func main() {
	service := flag.String("service", "", "Service to run: ride|driver|auth|admin")
	flag.Parse()

	// Allow service to be specified via environment variable
	if *service == "" {
		*service = os.Getenv("SERVICE")
	}

	switch *service {
	case "ride":
		runRideService()
	case "driver":
		runDriverService()
	case "auth":
		runAuthService()
	case "admin":
		runAdminService()
	default:
		fmt.Println("Usage: ride-hail-system -service=[ride|driver|auth|admin]")
		fmt.Println("   or: SERVICE=ride ride-hail-system")
		os.Exit(1)
	}
}

func runRideService() {
	log := util.New()

	log.Info("RideService", "Starting service initialization...")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Config", "Failed to load configuration", err)
	}
	log.OK("Config", "Configuration loaded successfully")

	database := db.ConnectToDB(&cfg.Database)
	if database == nil {
		log.Fatal("Database", "Failed to connect to database", err)
	}
	defer database.Close()
	log.OK("Database", "Connected successfully")

	rmqConn, rmqCh, err := mq.ConnectToRMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatal("RabbitMQ", "Failed to connect to RabbitMQ", err)
	}
	defer rmqConn.Close()
	defer rmqCh.Close()
	log.OK("RabbitMQ", "Connected successfully")

	publisher := mq.NewPublisher(rmqCh)
	repository := rideRepo.NewRideRepo(database)

	service := rideApp.NewRideService(repository, publisher, log)
	handler := rideApi.NewHandler(service)

	// Get WebSocket manager for real-time notifications
	wsManager := rideApi.GetGlobalWSManager()

	consumerInstance := consumer.NewDriverResponseConsumer(service, rmqCh, wsManager)
	if err := consumerInstance.Start(context.Background()); err != nil {
		log.Fatal("DriverResponseConsumer", "Failed to start driver response consumer", err)
	}

	log.OK("DriverResponseConsumer", "Started successfully")

	// Start location updates consumer
	locationConsumer := consumer.NewLocationConsumer(service, rmqCh, wsManager)
	if err := locationConsumer.Start(context.Background()); err != nil {
		log.Fatal("LocationConsumer", "Failed to start location consumer", err)
	}
	log.OK("LocationConsumer", "Started successfully")

	// Start ride status updates consumer
	rideStatusConsumer := consumer.NewRideStatusConsumer(service, rmqCh, wsManager)
	if err := rideStatusConsumer.Start(context.Background()); err != nil {
		log.Fatal("RideStatusConsumer", "Failed to start ride status consumer", err)
	}
	log.OK("RideStatusConsumer", "Started successfully")

	mux := handler.RegisterRoutes(repository)

	server := &http.Server{
		Addr:    ":" + "3000",
		Handler: mux,
	}

	go func() {
		log.OK("HTTP", "ride-service running on :3000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP", "Server error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("RideService", "Shutting down ride-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("HTTP", "Shutdown error", err)
	} else {
		log.OK("HTTP", "Server stopped gracefully")
	}
	log.Info("RideService", "Shutdown complete")
}

func runDriverService() {
	log := util.New()

	log.Info("DriverService", "Starting service initialization...")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Config", "Failed to load configuration", err)
	}
	log.OK("Config", "Configuration loaded successfully")

	database := db.ConnectToDB(&cfg.Database)
	if database == nil {
		log.Fatal("Database", "Failed to connect to database", err)
	}
	defer database.Close()
	log.OK("Database", "Connected successfully")

	// Connect to RabbitMQ
	rmqConn, rmqCh, err := mq.ConnectToRMQ(&cfg.RabbitMQ)
	if err != nil {
		log.Fatal("RabbitMQ", "Failed to connect to RabbitMQ", err)
	}
	defer rmqConn.Close()
	defer rmqCh.Close()
	log.OK("RabbitMQ", "Connected successfully")

	repo := psql.NewRepo(database)
	broker := driverRmq.NewBroker(rmqCh)
	service := usecase.NewService(repo, broker)
	handler := handlers.NewHandler(service)

	// Start driver matching consumer with WebSocket manager
	wsManager := handler.GetWSManager()
	consumer := usecase.NewMatchingConsumer(service, repo, rmqCh, wsManager)
	if err := consumer.Start(); err != nil {
		log.Fatal("MatchingConsumer", "Failed to start matching consumer", err)
	}
	log.OK("MatchingConsumer", "Started successfully")

	// Connect handler to matching consumer for processing driver responses
	handler.SetMatchingConsumer(consumer)

	mux := handler.Router()

	server := &http.Server{
		Addr:    ":" + "3001",
		Handler: mux,
	}

	go func() {
		log.OK("HTTP", "driver-service running on :3001")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP", "Server error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("DriverService", "Shutting down driver-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("HTTP", "Shutdown error", err)
	} else {
		log.OK("HTTP", "Server stopped gracefully")
	}
	log.Info("DriverService", "Shutdown complete")
}

func runAuthService() {
	log := util.New()

	log.Info("AuthService", "Starting service initialization...")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Config", "Failed to load configuration", err)
	}
	log.OK("Config", "Configuration loaded successfully")

	dbConn := db.ConnectToDB(&cfg.Database)
	if dbConn == nil {
		log.Fatal("Database", "Failed to connect to database", err)
	}
	defer dbConn.Close()
	log.OK("Database", "Connected successfully")

	repository := authRepo.NewAuthRepo(dbConn)
	service := authApp.NewAuthService(repository, log)
	handler := api.NewHandler(service)

	mux := handler.RegisterRoutes()

	server := &http.Server{
		Addr:    ":4000",
		Handler: mux,
	}

	go func() {
		log.OK("HTTP", "auth-service running on :4000")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP", "Server error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("AuthService", "Shutting down auth-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("HTTP", "Shutdown error", err)
	} else {
		log.OK("HTTP", "Server stopped gracefully")
	}

	log.Info("AuthService", "Shutdown complete")
}

func runAdminService() {
	log := util.New()

	log.Info("AdminService", "Starting service initialization...")

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Config", "Failed to load configuration", err)
	}
	log.OK("Config", "Configuration loaded successfully")

	dbConn := db.ConnectToDB(&cfg.Database)
	if dbConn == nil {
		log.Fatal("Database", "Failed to connect to database", err)
	}
	defer dbConn.Close()
	log.OK("Database", "Connected successfully")

	repository := adminRepo.NewAdminRepo(dbConn)
	service := adminApp.NewAdminService(repository)
	handler := adminApi.NewHandler(service)

	mux := handler.RegisterRoutes()

	server := &http.Server{
		Addr:    ":3004",
		Handler: mux,
	}

	go func() {
		log.OK("HTTP", "admin-service running on :3004")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP", "Server error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Warn("AdminService", "Shutting down admin-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("HTTP", "Shutdown error", err)
	} else {
		log.OK("HTTP", "Server stopped gracefully")
	}

	log.Info("AdminService", "Shutdown complete")
}
