package psql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ride-hail/internal/driver/models"
	"ride-hail/internal/ride/domain"
	"ride-hail/internal/shared/util"

	"github.com/jackc/pgx/v5"
)

func (r *repo) UpdateCurrLocation(ctx context.Context, data *models.LocalHistory, update bool) (*models.Coordinate, error) {
	insertCoordinates := `INSERT INTO coordinates(entity_id, entity_type, address, latitude, longitude, fare_amount, distance_km, duration_minutes) VALUES ($1, $2, 'Unknown', $3, $4, 0, 0, 0) RETURNING id, updated_at;`
	updatePrevCoordinates := `UPDATE coordinates SET is_current = false, updated_at = now() WHERE entity_id = $1 AND entity_type = $2 AND is_current = true`
	insertLocalHist := `INSERT INTO location_history(coordinate_id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees) VALUES ($1, $2, $3, $4, $5, $6, $7);`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	result := &models.Coordinate{}

	_, err = r.db.Exec(ctx, updatePrevCoordinates, data.DriverID, "driver")

	err = r.db.QueryRow(ctx, insertCoordinates, data.DriverID, "driver", data.Latitude, data.Longitude).Scan(&result.CoordinateID, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}

	_, err = r.db.Exec(ctx, insertLocalHist, result.CoordinateID, data.DriverID, data.Latitude, data.Longitude, data.AccuracyMeters, data.SpeedKmh, data.HeadingDegrees)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repo) StartRide(ctx context.Context, rideID, driverID, address string, driverLocation models.Location, accuracy, speed, heading *float64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Mark previous current location as not current
	_, err = tx.Exec(ctx, `
		UPDATE coordinates 
		SET is_current = false 
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
	`, driverID)
	if err != nil {
		return err
	}

	// Insert new current location
	var coordinateID string
	err = tx.QueryRow(ctx, `
		INSERT INTO coordinates (entity_id, entity_type, latitude, longitude, is_current, address)
		VALUES ($1, 'driver', $2, $3, true, $4)
		RETURNING id
	`, driverID, driverLocation.Latitude, driverLocation.Longitude, address).Scan(&coordinateID)
	if err != nil {
		return err
	}

	// Add to location history
	_, err = tx.Exec(ctx, `
		INSERT INTO location_history (coordinate_id, driver_id, latitude, longitude, 
		                            accuracy_meters, speed_kmh, heading_degrees)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, coordinateID, driverID, driverLocation.Latitude, driverLocation.Longitude, accuracy, speed, heading)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE drivers
		SET status = 'BUSY',
		    updated_at = NOW()
		WHERE id = $1
	`, driverID)
	if err != nil {
		return fmt.Errorf("failed updating driver: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

func (r *repo) CompleteRide(ctx context.Context, rideID, driverID, address string, finalLocation models.Location, actualDistanceKM float64, actualDurationMinutes int) (*float64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get ride details and verify driver ownership
	ride, err := r.getRideForCompletion(ctx, tx, rideID, driverID)
	if err != nil {
		return nil, err
	}

	// Calculate final fare
	finalFare := util.CalculateFinalFare(ride.RideType, actualDistanceKM, float64(actualDurationMinutes))
	driverEarnings := util.CalculateDriverEarnings(finalFare, ride.RideType)

	// Store final destination coordinate
	coordID, err := r.storeFinalCoordinate(ctx, tx, driverID, finalLocation, finalFare, actualDistanceKM, actualDurationMinutes)
	if err != nil {
		return nil, err
	}

	// Update ride status and final details
	completedAt := time.Now().UTC()
	err = r.updateRideCompletion(ctx, tx, rideID, finalFare, driverEarnings, coordID, completedAt, actualDistanceKM, actualDurationMinutes)
	if err != nil {
		return nil, err
	}

	// Update driver status and earnings
	err = r.updateDriverAfterCompletion(ctx, tx, driverID, driverEarnings)
	if err != nil {
		return nil, err
	}

	// Record ride completion event
	err = r.recordRideCompletionEvent(ctx, tx, rideID, finalFare, driverEarnings, actualDistanceKM, actualDurationMinutes)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &driverEarnings, nil
}

func (r *repo) getRideForCompletion(ctx context.Context, tx pgx.Tx, rideID, driverID string) (*domain.Ride, error) {
	const query = `
		SELECT r.id, r.passenger_id, r.driver_id, r.vehicle_type, r.status, 
		       r.estimated_fare, r.final_fare, r.pickup_coordinate_id,
		       c.latitude as pickup_lat, c.longitude as pickup_lng
		FROM rides r
		LEFT JOIN coordinates c ON r.pickup_coordinate_id = c.id
		WHERE r.id = $1 AND r.driver_id = $2 AND r.status = 'IN_PROGRESS'
		FOR UPDATE
	`

	var ride domain.Ride
	err := tx.QueryRow(ctx, query, rideID, driverID).Scan(
		&ride.ID, &ride.PassengerID, &ride.DriverID, &ride.RideType, &ride.Status,
		&ride.EstimatedFare, &ride.FinalFare, &ride.PickupCoordinateID,
		&ride.PickupLat, &ride.PickupLng,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("ride not found or not in progress for this driver")
		}
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	return &ride, nil
}

// storeFinalCoordinate stores the final destination coordinate
func (r *repo) storeFinalCoordinate(ctx context.Context, tx pgx.Tx, driverID string, location models.Location, fare, distanceKM float64, durationMinutes int) (string, error) {
	const query = `
		INSERT INTO coordinates (
			id, created_at, updated_at, entity_id, entity_type, 
			address, latitude, longitude, fare_amount, 
			distance_km, duration_minutes, is_current
		) VALUES (
			gen_random_uuid(), NOW(), NOW(), $1, 'driver',
			$2, $3, $4, $5, $6, $7, true
		) RETURNING id
	`

	var coordID string
	err := tx.QueryRow(ctx, query,
		driverID,
		"Destination Location", // In production, you'd geocode this
		location.Latitude,
		location.Longitude,
		fare,
		distanceKM,
		durationMinutes,
	).Scan(&coordID)
	if err != nil {
		return "", fmt.Errorf("failed to store final coordinate: %w", err)
	}

	return coordID, nil
}

// updateRideCompletion updates the ride record with completion details
func (r *repo) updateRideCompletion(ctx context.Context, tx pgx.Tx, rideID string, finalFare, driverEarnings float64, coordID string, completedAt time.Time, distanceKM float64, durationMinutes int) error {
	const query = `
		UPDATE rides 
		SET status = 'COMPLETED',
		    final_fare = $2,
		    completed_at = $3,
		    destination_coordinate_id = $4,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := tx.Exec(ctx, query, rideID, finalFare, completedAt, coordID)
	if err != nil {
		return fmt.Errorf("failed to update ride completion: %w", err)
	}

	return nil
}

// updateDriverAfterCompletion updates driver status and earnings
func (r *repo) updateDriverAfterCompletion(ctx context.Context, tx pgx.Tx, driverID string, driverEarnings float64) error {
	const query = `
		UPDATE drivers 
		SET status = 'AVAILABLE',
		    total_rides = total_rides + 1,
		    total_earnings = total_earnings + $2,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := tx.Exec(ctx, query, driverID, driverEarnings)
	if err != nil {
		return fmt.Errorf("failed to update driver after completion: %w", err)
	}

	return nil
}

// recordRideCompletionEvent records the completion event in ride_events table
func (r *repo) recordRideCompletionEvent(ctx context.Context, tx pgx.Tx, rideID string, finalFare, driverEarnings float64, distanceKM float64, durationMinutes int) error {
	const query = `
		INSERT INTO ride_events (
			id, created_at, ride_id, event_type, event_data
		) VALUES (
			gen_random_uuid(), NOW(), $1, 'RIDE_COMPLETED', $2
		)
	`

	eventData := map[string]interface{}{
		"final_fare":              finalFare,
		"driver_earnings":         driverEarnings,
		"actual_distance_km":      distanceKM,
		"actual_duration_minutes": durationMinutes,
		"completed_at":            time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	_, err = tx.Exec(ctx, query, rideID, jsonData)
	if err != nil {
		return fmt.Errorf("failed to record ride completion event: %w", err)
	}

	return nil
}
