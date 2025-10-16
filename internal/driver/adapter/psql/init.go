package psql

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"ride-hail/internal/driver/models"
)

type repo struct {
	db *pgxpool.Pool
}

type Repo interface {
	CreateSessionDriver(ctx context.Context, data models.Location) (string, error)
	FinishSession(ctx context.Context, id string) error
	UpdateCurrLocation(ctx context.Context, data *models.LocalHistory, update bool) (*models.Coordinate, error)
	CheckDriverExists(ctx context.Context, driverID string) error
}

func NewRepo(db *pgxpool.Pool) Repo {
	return &repo{db: db}
}
