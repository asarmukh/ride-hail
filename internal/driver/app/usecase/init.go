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
	StartSession(ctx context.Context, data models.LocationHistory) (string, error)
}

func NewService(repo psql.Repo, broker rmq.Broker) Service {
	return &service{repo, broker}
}
