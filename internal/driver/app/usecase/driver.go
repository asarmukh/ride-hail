package usecase

import (
	"context"
	"errors"
	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/apperrors"
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

func (s *service) FinishSession(ctx context.Context, id string) error {
	err := s.repo.CheckDriverExists(ctx, id)
	if !errors.Is(err, apperrors.ErrDriverOnline) {
		return err
	}

	return s.repo.FinishSession(ctx, id)
}
