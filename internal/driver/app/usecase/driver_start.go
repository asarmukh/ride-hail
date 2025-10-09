package usecase

import (
	"context"
	"ride-hail/internal/driver/models"
)

func (s *service) StartSession(ctx context.Context, data models.LocationHistory) (string, error) {
	if err := s.repo.CheckDriverExists(ctx, data.DriverID); err != nil {
		return "", err
	}

	id, err := s.repo.CreateSessionDriver(ctx, data)
	if err != nil {
		return "", err
	}

	return id, nil
}
