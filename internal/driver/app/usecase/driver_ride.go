package usecase

import (
	"context"
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
	err := util.ValidateLocation(driverLocation)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// I guess address should be calculated using location
	address := "dummy" // DUMMY

	err = s.repo.StartRide(ctx, rideID, driverID, address, driverLocation, nil, nil, nil)
	if err != nil {
		return http.StatusBadGateway, err
	}

	return http.StatusOK, nil
}

// ctx context.Context, rideID, driverID, address string, finalLocation models.Location, actualDistanceKM float64, actualDurationMinutes int
func (s *service) CompleteRide(ctx context.Context, driverID, rideID string, finalLocation models.Location, distance float64, duration int) (float64, int, error) {
	err := util.ValidateCompleteRideRequest(finalLocation, distance, duration)
	if err != nil {
		return 0, http.StatusBadRequest, err
	}

	// I guess address should be calculated using location
	address := "dummy" // DUMMY
	// Update driver status back to available
	driverEarnings, err := s.repo.CompleteRide(ctx, rideID, driverID, address, finalLocation, distance, duration)
	if err != nil {
		return 0, http.StatusBadGateway, err
	}
	// Update driver stats (simplified)
	// In real implementation, calculate earnings and update total rides

	// Publish status update
	// if err := s.rabbitMQ.PublishDriverStatus(driverID, models.DriverAvailable, rideID); err != nil {
	// 	return err
	// }

	return *driverEarnings, http.StatusOK, nil
}
