package main

import (
	"net/http"
	"ride-hail/internal/driver/adapter/handlers"
	"ride-hail/internal/driver/adapter/psql"
	"ride-hail/internal/driver/app/usecase"
	"ride-hail/internal/shared/config"
	"ride-hail/internal/shared/db"
	"ride-hail/internal/shared/mq"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		panic(err)
	}

	conn, ch, err := mq.ConnectToRMQ(&cfg.RabbitMQ)
	if err != nil {
		panic(err)
	}

	defer conn.Close()
	defer ch.Close()

	database := db.ConnectToDB(&cfg.Database)

	repo := psql.NewRepo(database)
	service := usecase.NewService(repo, ch)
	handler := handlers.NewHandler(service)

	mux := handler.Router()

	server := &http.Server{
		Addr:    ":3001",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
}
