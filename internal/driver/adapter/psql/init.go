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
	CreateSessionDriver(ctx context.Context, data models.LocationHistory) (string, error)
	FinishSession(ctx context.Context, id string) error
	CheckDriverExists(ctx context.Context, driverID string) error
}

func NewRepo(db *pgxpool.Pool) Repo {
	return &repo{db: db}
}
