package psql

import (
	"context"

	"ride-hail/internal/driver/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repo struct {
	db *pgxpool.Pool
}

type Repo interface {
	// CreateSessionDriver(ctx context.Context, data models.Location) (string, error)
	FinishSession(ctx context.Context, driverID string) (*models.FinishDriverResponse, error)
	UpdateCurrLocation(ctx context.Context, data *models.LocalHistory, update bool) (*models.Coordinate, error)
	// CheckDriverExists(ctx context.Context, driverID string) error
	StartDriverSession(ctx context.Context, driverID string, location models.Location) (string, error)
	CheckUserExistsAndIsDriver(ctx context.Context, userID string) error
	StartRide(ctx context.Context, rideID, driverID, address string, driverLocation models.Location, accuracy, speed, heading *float64) error
	CompleteRide(ctx context.Context, rideID, driverID, address string, finalLocation models.Location, actualDistanceKM float64, actualDurationMinutes int) (*float64, error)
}

func NewRepo(db *pgxpool.Pool) Repo {
	return &repo{db: db}
}
