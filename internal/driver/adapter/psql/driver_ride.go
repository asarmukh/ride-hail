package psql

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get ride details and verify it's ready to start
	ride, err := r.getRideForStart(ctx, tx, rideID, driverID)
	if err != nil {
		return err
	}

	// Verify driver is at pickup location (with some tolerance)
	if err := r.verifyDriverAtPickup(driverLocation, ride.PickupLat, ride.PickupLng); err != nil {
		return err
	}

	// Update driver status to BUSY
	if err := r.updateDriverStatus(ctx, tx, driverID, "BUSY"); err != nil {
		return err
	}

	// Update ride status to IN_PROGRESS
	startedAt := time.Now().UTC()
	if err := r.updateRideStart(ctx, tx, rideID, startedAt); err != nil {
		return err
	}

	// Record ride start event
	if err := r.recordRideStartEvent(ctx, tx, rideID, driverLocation); err != nil {
		return err
	}

	// Store current driver location
	if err := r.storeDriverLocation(ctx, tx, driverID, driverLocation, rideID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// getRideForStart retrieves and validates ride for starting
func (h *repo) getRideForStart(ctx context.Context, tx pgx.Tx, rideID, driverID string) (*domain.Ride, error) {
	const query = `
		SELECT 
			r.id, r.passenger_id, r.driver_id, r.vehicle_type, r.status, 
			r.pickup_coordinate_id, c.latitude as pickup_lat, c.longitude as pickup_lng,
			d.status as driver_status
		FROM rides r
		LEFT JOIN coordinates c ON r.pickup_coordinate_id = c.id
		LEFT JOIN drivers d ON r.driver_id = d.id
		WHERE r.id = $1 AND r.driver_id = $2
	`

	var ride domain.Ride
	var driverStatus string
	var pickupLat, pickupLng *float64

	err := tx.QueryRow(ctx, query, rideID, driverID).Scan(
		&ride.ID, &ride.PassengerID, &ride.DriverID, &ride.RideType, &ride.Status,
		&ride.PickupCoordinateID, &pickupLat, &pickupLng, &driverStatus,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("ride not found or driver not assigned to this ride")
		}
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	// Handle nullable coordinates
	if pickupLat != nil {
		ride.PickupLat = *pickupLat
	}
	if pickupLng != nil {
		ride.PickupLng = *pickupLng
	}

	// Validate ride can be started
	if driverStatus != "EN_ROUTE" {
		return nil, fmt.Errorf("driver status must be EN_ROUTE to start ride. current status: %s", driverStatus)
	}

	return &ride, nil
}

// verifyDriverAtPickup verifies driver is at the pickup location (within 100 meters)
func (h *repo) verifyDriverAtPickup(driverLocation models.Location, pickupLat, pickupLng float64) error {
	// Check if we have valid pickup coordinates
	if pickupLat == 0 && pickupLng == 0 {
		return fmt.Errorf("invalid pickup coordinates")
	}

	// Calculate distance between driver and pickup location
	distance := h.calculateDistance(
		driverLocation.Latitude, driverLocation.Longitude,
		pickupLat, pickupLng,
	)

	// Allow 100 meters tolerance
	if distance > 0.1 { // 100 meters in kilometers
		return fmt.Errorf("driver is too far from pickup location: %.2f meters away", distance*1000)
	}

	return nil
}

// calculateDistance calculates distance between two points using Haversine formula
func (h *repo) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Differences
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	// Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// updateRideStart updates the ride record with start details
func (h *repo) updateRideStart(ctx context.Context, tx pgx.Tx, rideID string, startedAt time.Time) error {
	const query = `
		UPDATE rides 
		SET status = 'IN_PROGRESS',
		    started_at = $2,
		    updated_at = NOW(),
			arrived_at = NOW()
		WHERE id = $1
	`

	_, err := tx.Exec(ctx, query, rideID, startedAt)
	if err != nil {
		return fmt.Errorf("failed to update ride start: %w", err)
	}

	return nil
}

// recordRideStartEvent records the start event in ride_events table
func (h *repo) recordRideStartEvent(ctx context.Context, tx pgx.Tx, rideID string, driverLocation models.Location) error {
	const query = `
		INSERT INTO ride_events (
			id, created_at, ride_id, event_type, event_data
		) VALUES (
			gen_random_uuid(), NOW(), $1, 'RIDE_STARTED', $2
		)
	`

	eventData := map[string]interface{}{
		"started_at": time.Now().UTC().Format(time.RFC3339),
		"driver_location": map[string]float64{
			"lat": driverLocation.Latitude,
			"lng": driverLocation.Longitude,
		},
	}

	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	_, err = tx.Exec(ctx, query, rideID, jsonData)
	if err != nil {
		return fmt.Errorf("failed to record ride start event: %w", err)
	}

	return nil
}

// storeDriverLocation stores the current driver location
func (h *repo) storeDriverLocation(ctx context.Context, tx pgx.Tx, driverID string, location models.Location, rideID string) error {
	// First, mark previous current location as not current
	const updatePreviousQuery = `
		UPDATE coordinates 
		SET is_current = false, updated_at = NOW()
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
	`

	_, err := tx.Exec(ctx, updatePreviousQuery, driverID)
	if err != nil {
		return fmt.Errorf("failed to update previous location: %w", err)
	}

	// Insert new current location
	const insertQuery = `
		INSERT INTO coordinates (
			id, created_at, updated_at, entity_id, entity_type, 
			address, latitude, longitude, is_current
		) VALUES (
			gen_random_uuid(), NOW(), NOW(), $1, 'driver',
			'Ride in progress', $2, $3, true
		)
		RETURNING id
	`

	var coordID string
	err = tx.QueryRow(ctx, insertQuery, driverID, location.Latitude, location.Longitude).Scan(&coordID)
	if err != nil {
		return fmt.Errorf("failed to store driver location: %w", err)
	}

	// Also record in location_history for tracking during ride
	const historyQuery = `
		INSERT INTO location_history (
			id, coordinate_id, driver_id, latitude, longitude,
			recorded_at, ride_id
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, NOW(), $5
		)
	`

	_, err = tx.Exec(ctx, historyQuery, coordID, driverID, location.Latitude, location.Longitude, rideID)
	if err != nil {
		return fmt.Errorf("failed to record location history: %w", err)
	}

	return nil
}

// -------------------------------------------------------------------------------------------------------------------------------------

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
		JOIN coordinates c ON r.pickup_coordinate_id = c.id
		WHERE r.id = $1 AND r.driver_id = $2 AND r.status = 'IN_PROGRESS'
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
		return nil, err
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
