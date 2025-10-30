package usecase

import (
	"context"
	"sync"
	"time"

	"ride-hail/internal/driver/adapter/psql"
	"ride-hail/internal/driver/adapter/rmq"
	"ride-hail/internal/driver/models"
)

type service struct {
	repo        psql.Repo
	broker      rmq.Broker
	rateLimiter *RateLimiter
}

type Service interface {
	RegisterDriver(ctx context.Context, driverData *models.Driver) (int, error)
	StartSession(ctx context.Context, driverID string, location models.Location) (string, error)
	FinishSession(ctx context.Context, driverID string) (*models.FinishDriverResponse, error)
	UpdateLocation(ctx context.Context, data *models.LocalHistory) (*models.Coordinate, error)
	UpdateDriverStatus(ctx context.Context, driverID interface{}, status string) error
	UpdateDriverLocation(ctx context.Context, driverID interface{}, lat, lng, accuracy, speed, heading float64) error
	CanUpdateLocation(driverID string) bool
	StartRide(ctx context.Context, rideID, driverID string, lat, lng float64) error
	CompleteRide(ctx context.Context, rideID, driverID string, lat, lng float64, actualDistanceKm float64, actualDurationMin int) (float64, error)
}

type RateLimiter struct {
	lastUpdate map[string]time.Time
	mu         sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		lastUpdate: make(map[string]time.Time),
	}
}

func (rl *RateLimiter) Allow(driverID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	last, exists := rl.lastUpdate[driverID]
	if exists && time.Since(last) < 3*time.Second {
		return false
	}

	rl.lastUpdate[driverID] = time.Now()
	return true
}

func NewService(repo psql.Repo, broker rmq.Broker) Service {
	return &service{
		repo:        repo,
		broker:      broker,
		rateLimiter: NewRateLimiter(),
	}
}
