package psql

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

type repo struct {
	db *pgxpool.Pool
}

type Repo interface{}

func NewRepo(db *pgxpool.Pool) Repo {
	return &repo{db: db}
}
