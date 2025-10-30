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

func (s *service) FinishSession(ctx context.Context, driverID string) (*models.FinishDriverResponse, error) {
	response, err := s.repo.FinishSession(ctx, driverID)
	if err != nil {
		return nil, err
	}
	return response, nil
}
