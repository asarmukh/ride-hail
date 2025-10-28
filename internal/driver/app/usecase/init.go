package usecase

import (
	"context"

	"ride-hail/internal/driver/adapter/psql"
	"ride-hail/internal/driver/adapter/rmq"
	"ride-hail/internal/driver/models"
)

type service struct {
	repo   psql.Repo
	broker rmq.Broker
}

type Service interface {
	RegisterDriver(ctx context.Context, driverData *models.Driver) (int, error)
	StartSession(ctx context.Context, data models.Location) (string, error)
	FinishSession(ctx context.Context, id string) error
	UpdateLocation(ctx context.Context, data *models.LocalHistory) (*models.Coordinate, error)
	StartRide(ctx context.Context, rideID, driverID string, driverLocation models.Location) (int, error)
	CompleteRide(ctx context.Context, driverID, rideID string, finalLocation models.Location, distance float64, duration int) (float64, int, error)
}

func NewService(repo psql.Repo, broker rmq.Broker) Service {
	return &service{repo, broker}
}
