package psql

import (
	"context"

	"ride-hail/internal/driver/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type repo struct {
	db *pgxpool.Pool
}

type NearbyDriver struct {
	ID             string
	Email          string
	Rating         float64
	TotalRides     int
	CompletedRides int
	VehicleType    string
	VehicleAttrs   map[string]interface{}
	Latitude       float64
	Longitude      float64
	DistanceKm     float64
}

type Repo interface {
	InsertDriver(ctx context.Context, driverData *models.Driver) error
	// CreateSessionDriver(ctx context.Context, data models.Location) (string, error)
	FinishSession(ctx context.Context, driverID string) (*models.FinishDriverResponse, error)
	UpdateCurrLocation(ctx context.Context, data *models.LocalHistory, update bool) (*models.Coordinate, error)
	// CheckDriverExists(ctx context.Context, driverID string) error
	CheckUserExistsAndIsDriver(ctx context.Context, userID string) error
	FindNearbyDrivers(ctx context.Context, lat, lng float64, vehicleType string, radiusKm float64) ([]NearbyDriver, error)
	UpdateDriverStatus(ctx context.Context, driverID string, status string) error
	StartDriverSession(ctx context.Context, driverID string, location models.Location) (string, error)
	CheckLicenseNumberExists(ctx context.Context, licenseNumber string) (bool, error)
}

func NewRepo(db *pgxpool.Pool) Repo {
	return &repo{db: db}
}
