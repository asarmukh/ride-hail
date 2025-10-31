package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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

	coord, err := s.repo.UpdateCurrLocation(ctx, data, update)
	if err != nil {
		return nil, err
	}

	// Publish location update to location_fanout exchange
	locationUpdate := map[string]interface{}{
		"driver_id": data.DriverID,
		"ride_id":   data.RideID,
		"location": map[string]float64{
			"lat": data.Latitude,
			"lng": data.Longitude,
		},
		"speed_kmh":       data.SpeedKmh,
		"heading_degrees": data.HeadingDegrees,
		"accuracy_meters": data.AccuracyMeters,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"coordinate_id":   coord.CoordinateID,
	}

	if err := s.broker.PublishFanout(ctx, "location_fanout", locationUpdate); err != nil {
		// Log error but don't fail the request
		slog.Error("Failed to broadcast location", "error", err, "driver_id", data.DriverID)
	}

	return coord, nil
}

func (s *service) CanUpdateLocation(driverID string) bool {
	return s.rateLimiter.Allow(driverID)
}

// UpdateDriverStatus updates the driver's status in the database
func (s *service) UpdateDriverStatus(ctx context.Context, driverID string, status string) error {
	return s.repo.UpdateDriverStatus(ctx, driverID, status)
}

// UpdateDriverLocation updates the driver's location via WebSocket
func (s *service) UpdateDriverLocation(ctx context.Context, driverID string, lat, lng, accuracy, speed, heading float64) error {
	// Create location history record
	locationHistory := &models.LocalHistory{
		DriverID:       driverID,
		Latitude:       lat,
		Longitude:      lng,
		AccuracyMeters: accuracy,
		SpeedKmh:       speed,
		HeadingDegrees: heading,
	}

	coord, err := s.repo.UpdateCurrLocation(ctx, locationHistory, true)
	if err != nil {
		return err
	}

	// Publish location update to location_fanout exchange
	locationUpdate := map[string]interface{}{
		"driver_id": driverID,
		"ride_id":   locationHistory.RideID,
		"location": map[string]float64{
			"lat": lat,
			"lng": lng,
		},
		"speed_kmh":       speed,
		"heading_degrees": heading,
		"accuracy_meters": accuracy,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"coordinate_id":   coord.CoordinateID,
	}

	if err := s.broker.PublishFanout(ctx, "location_fanout", locationUpdate); err != nil {
		// Log error but don't fail the request
		slog.Error("Failed to broadcast location", "error", err, "driver_id", driverID)
	}

	return nil
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

// CompleteRide transitions a ride to COMPLETED status and calculates final fare
func (s *service) CompleteRide(ctx context.Context, rideID, driverID string, finalLocation models.Location, actualDistanceKm float64, actualDurationMin int) (float64, int, error) {
	err := util.ValidateCompleteRideRequest(finalLocation, actualDistanceKm, actualDurationMin)
	if err != nil {
		return 0, http.StatusBadRequest, err
	}
	fmt.Println(rideID, driverID)
	// I guess address should be calculated using location
	address := "dummy" // DUMMY
	// Update driver status back to available
	driverEarnings, err := s.repo.CompleteRide(ctx, rideID, driverID, address, finalLocation, actualDistanceKm, actualDurationMin)
	if err != nil {
		return 0, http.StatusBadGateway, err
	}

	completionUpdate := map[string]interface{}{
		"ride_id":             rideID,
		"driver_id":           driverID,
		"status":              "COMPLETED",
		"completed_at":        time.Now().UTC().Format(time.RFC3339),
		"final_fare":          driverEarnings,
		"actual_distance_km":  actualDistanceKm,
		"actual_duration_min": actualDurationMin,
		"final_location": map[string]float64{
			"lat": finalLocation.Latitude,
			"lng": finalLocation.Longitude,
		},
	}

	if err := s.broker.Publish(ctx, "ride_topic", "ride.status.completed", completionUpdate); err != nil {
		slog.Error("Failed to publish ride completion event", "error", err, "ride_id", rideID)
		return 0, 0, fmt.Errorf("failed to publish ride completion event: %w", err)
	}

	// Update driver status to AVAILABLE
	if err := s.UpdateDriverStatus(ctx, driverID, "AVAILABLE"); err != nil {
		slog.Error("Failed to update driver status", "error", err, "driver_id", driverID)
		return 0, 0, fmt.Errorf("failed to update driver status: %w", err)
	}

	slog.Info("Ride completed", "ride_id", rideID, "driver_id", driverID, "final_fare", driverEarnings)
	return *driverEarnings, http.StatusOK, nil
}
