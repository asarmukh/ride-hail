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
	CheckDriverExists(ctx context.Context, driverID string) error
	CreateSessionDriver(ctx context.Context, data models.LocationHistory) (string, error)
}

func NewRepo(db *pgxpool.Pool) Repo {
	return &repo{db: db}
}
