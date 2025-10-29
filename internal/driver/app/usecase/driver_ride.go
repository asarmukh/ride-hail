package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ride-hail/internal/driver/models"

	"github.com/google/uuid"
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
func (s *service) UpdateDriverStatus(ctx context.Context, driverID interface{}, status string) error {
	var driverIDStr string

	switch v := driverID.(type) {
	case uuid.UUID:
		driverIDStr = v.String()
	case string:
		// Validate it's a valid UUID
		if _, err := uuid.Parse(v); err != nil {
			return fmt.Errorf("invalid driver ID: %w", err)
		}
		driverIDStr = v
	default:
		return fmt.Errorf("unsupported driver ID type: %T", driverID)
	}

	return s.repo.UpdateDriverStatus(ctx, driverIDStr, status)
}

// UpdateDriverLocation updates the driver's location via WebSocket
func (s *service) UpdateDriverLocation(ctx context.Context, driverID interface{}, lat, lng, accuracy, speed, heading float64) error {
	var driverIDStr string

	switch v := driverID.(type) {
	case uuid.UUID:
		driverIDStr = v.String()
	case string:
		// Validate it's a valid UUID
		if _, err := uuid.Parse(v); err != nil {
			return fmt.Errorf("invalid driver ID: %w", err)
		}
		driverIDStr = v
	default:
		return fmt.Errorf("unsupported driver ID type: %T", driverID)
	}

	// Create location history record
	locationHistory := &models.LocalHistory{
		DriverID:       driverIDStr,
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
		"driver_id": driverIDStr,
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
		slog.Error("Failed to broadcast location", "error", err, "driver_id", driverIDStr)
	}

	return nil
}

// StartRide transitions a ride to IN_PROGRESS status
func (s *service) StartRide(ctx context.Context, rideID, driverID string, lat, lng float64) error {
	// Verify driver is assigned to this ride (could add DB check here)

	// Update ride status to IN_PROGRESS
	// Note: This would typically call a ride service endpoint or publish an event
	// For now, we'll publish to RabbitMQ
	statusUpdate := map[string]interface{}{
		"ride_id":    rideID,
		"driver_id":  driverID,
		"status":     "IN_PROGRESS",
		"started_at": time.Now().UTC().Format(time.RFC3339),
		"location": map[string]float64{
			"lat": lat,
			"lng": lng,
		},
	}

	if err := s.broker.Publish(ctx, "ride_topic", "ride.status.in_progress", statusUpdate); err != nil {
		slog.Error("Failed to publish ride start event", "error", err, "ride_id", rideID)
		return fmt.Errorf("failed to publish ride start event: %w", err)
	}

	// Update driver status to BUSY
	if err := s.UpdateDriverStatus(ctx, driverID, "BUSY"); err != nil {
		slog.Error("Failed to update driver status", "error", err, "driver_id", driverID)
		return fmt.Errorf("failed to update driver status: %w", err)
	}

	slog.Info("Ride started", "ride_id", rideID, "driver_id", driverID)
	return nil
}

// CompleteRide transitions a ride to COMPLETED status and calculates final fare
func (s *service) CompleteRide(ctx context.Context, rideID, driverID string, lat, lng float64, actualDistanceKm float64, actualDurationMin int) (float64, error) {
	// Calculate final fare based on actual distance and duration
	// Base fare calculation (this should match the fare calculation in ride service)
	baseFare := 200.0 // Base fare in KZT
	perKmRate := 100.0
	perMinRate := 50.0

	finalFare := baseFare + (actualDistanceKm * perKmRate) + (float64(actualDurationMin) * perMinRate)

	// Publish completion event
	completionUpdate := map[string]interface{}{
		"ride_id":             rideID,
		"driver_id":           driverID,
		"status":              "COMPLETED",
		"completed_at":        time.Now().UTC().Format(time.RFC3339),
		"final_fare":          finalFare,
		"actual_distance_km":  actualDistanceKm,
		"actual_duration_min": actualDurationMin,
		"final_location": map[string]float64{
			"lat": lat,
			"lng": lng,
		},
	}

	if err := s.broker.Publish(ctx, "ride_topic", "ride.status.completed", completionUpdate); err != nil {
		slog.Error("Failed to publish ride completion event", "error", err, "ride_id", rideID)
		return 0, fmt.Errorf("failed to publish ride completion event: %w", err)
	}

	// Update driver status to AVAILABLE
	if err := s.UpdateDriverStatus(ctx, driverID, "AVAILABLE"); err != nil {
		slog.Error("Failed to update driver status", "error", err, "driver_id", driverID)
		return 0, fmt.Errorf("failed to update driver status: %w", err)
	}

	slog.Info("Ride completed", "ride_id", rideID, "driver_id", driverID, "final_fare", finalFare)
	return finalFare, nil
}
