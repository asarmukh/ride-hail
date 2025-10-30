package usecase

import (
	"context"

	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/util"
)

func (s *service) StartSession(ctx context.Context, driverID string, driverLocation models.Location) (string, error) {
	if err := util.ValidateLocation(driverLocation); err != nil {
		return "", err
	}

	sessionID, err := s.repo.StartDriverSession(ctx, driverID, driverLocation)
	if err != nil {
		return "", err
	}

	// id, err := s.repo.CreateSessionDriver(ctx, location)
	// if err != nil {
	// 	return "", err
	// }

	return sessionID, nil
}

func (s *service) FinishSession(ctx context.Context, id string) error {
	// err := s.repo.CheckDriverExists(ctx, id)
	// if !errors.Is(err, apperrors.ErrDriverOnline) {
	// 	return err
	// }

	// return s.repo.FinishSession(ctx, id)
	return nil
}
