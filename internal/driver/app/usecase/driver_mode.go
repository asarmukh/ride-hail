package usecase

import (
	"context"

	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/util"
)

func (s *service) StartSession(ctx context.Context, location models.Location, driverID string) (string, error) {
	if err := util.ValidateLocation(location); err != nil {
		return "", err
	}

	sessionID, err := s.repo.StartDriverSession(ctx, driverID, location)
	if err != nil {
		return "", err
	}

	// id, err := s.repo.CreateSessionDriver(ctx, location)
	// if err != nil {
	// 	return "", err
	// }

	return sessionID, nil
}

func (s *service) FinishSession(ctx context.Context, id string) (*models.FinishDriverResponse, error) {
	response, err := s.repo.FinishSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return response, nil
}
