package usecase

import (
	"context"
	"fmt"
	"net/http"

	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/util"
)

func (s *service) UpdateLocation(ctx context.Context, data *models.LocalHistory) (*models.Coordinate, error) {
	update := true

	// if err := s.repo.CheckLocationExists(ctx, data.DriverID); errors.Is(err, pgx.ErrNoRows) {
	// 	update = false
	// } else if err != nil {
	// 	return nil, err
	// }

	return s.repo.UpdateCurrLocation(ctx, data, update)
}

func (s *service) StartRide(ctx context.Context, rideID, driverID string, driverLocation models.Location) (int, error) {
	if !util.LocationIsValid(driverLocation) {
		return http.StatusBadRequest, fmt.Errorf("latitude must be  between => 180 > x > -180; longitude must be between => 90 > x > -90")
	}

	err := s.repo.UpdateRide(ctx, rideID, driverID, driverLocation, nil, nil, nil)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("could not update ride: %v", err)
	}

	return http.StatusOK, nil
}
