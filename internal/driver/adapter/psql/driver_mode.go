package psql

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/apperrors"

	"github.com/jackc/pgx/v5"
)

func (r *repo) StartDriverSession(ctx context.Context, driverID string, location models.Location) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if driver exists and is verified
	err = r.getDriverForStart(ctx, tx, driverID)
	if err != nil {
		return "", err
	}

	// Check if driver already has an active session
	activeSession, err := r.getActiveDriverSession(ctx, tx, driverID)
	if err == nil && activeSession != nil {
		// Driver already has an active session, update status and return
		return r.handleExistingSession(ctx, tx, driverID, location, activeSession)
	}

	// Create new driver session
	sessionID, err := r.createDriverSession(ctx, tx, driverID)
	if err != nil {
		return "", err
	}

	// Update driver status to AVAILABLE
	if err := r.updateDriverStatus(ctx, tx, driverID, "AVAILABLE"); err != nil {
		return "", err
	}

	// Store initial driver location
	if err := r.storeInitialDriverLocation(ctx, tx, driverID, location); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return sessionID, nil
}

// getDriverForStart retrieves and validates driver for going online
func (r *repo) getDriverForStart(ctx context.Context, tx pgx.Tx, driverID string) error {
	const query = `
		SELECT d.id, d.license_number, d.vehicle_type, d.is_verified, d.status,
		       u.email, u.role, u.status as user_status
		FROM drivers d
		JOIN users u ON d.id = u.id
		WHERE d.id = $1
	`

	var driver models.Driver
	var userEmail, userRole, userStatus string
	err := tx.QueryRow(ctx, query, driverID).Scan(
		&driver.ID, &driver.LicenseNumber, &driver.VehicleType, &driver.IsVerified, &driver.Status,
		&userEmail, &userRole, &userStatus,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("driver not found")
		}
		return fmt.Errorf("failed to get driver: %w", err)
	}

	// Validate driver can go online
	if !driver.IsVerified {
		return fmt.Errorf("driver account is not verified")
	}

	if userStatus != "ACTIVE" {
		return fmt.Errorf("driver account is not active. status: %s", userStatus)
	}

	if userRole != "DRIVER" {
		return fmt.Errorf("user is not a driver")
	}

	return nil
}

// getActiveDriverSession checks if driver already has an active session
func (r *repo) getActiveDriverSession(ctx context.Context, tx pgx.Tx, driverID string) (*models.DriverSession, error) {
	const query = `
		SELECT id, driver_id, started_at, ended_at, total_rides, total_earnings
		FROM driver_sessions 
		WHERE driver_id = $1 AND ended_at IS NULL
		ORDER BY started_at DESC
		LIMIT 1
	`

	var session models.DriverSession
	err := tx.QueryRow(ctx, query, driverID).Scan(
		&session.ID, &session.DriverID, &session.StartedAt, &session.EndedAt,
		&session.TotalRides, &session.TotalEarnings,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No active session is fine
		}
		return nil, fmt.Errorf("failed to get driver session: %w", err)
	}

	return &session, nil
}

// handleExistingSession handles when driver already has an active session
func (r *repo) handleExistingSession(ctx context.Context, tx pgx.Tx, driverID string, location models.Location, session *models.DriverSession) (string, error) {
	// Update driver status to AVAILABLE
	if err := r.updateDriverStatus(ctx, tx, driverID, "AVAILABLE"); err != nil {
		return "", err
	}

	// Store current driver location
	if err := r.storeInitialDriverLocation(ctx, tx, driverID, location); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return session.ID, nil
}

// createDriverSession creates a new driver session
func (r *repo) createDriverSession(ctx context.Context, tx pgx.Tx, driverID string) (string, error) {
	const query = `
		INSERT INTO driver_sessions (
			id, driver_id, started_at, total_rides, total_earnings
		) VALUES (
			gen_random_uuid(), $1, NOW(), 0, 0
		) RETURNING id
	`

	var sessionID string
	err := tx.QueryRow(ctx, query, driverID).Scan(&sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to create driver session: %w", err)
	}

	return sessionID, nil
}

// updateDriverStatus updates driver status
func (r *repo) updateDriverStatus(ctx context.Context, tx pgx.Tx, driverID, status string) error {
	const query = `
		UPDATE drivers 
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`

	result, err := tx.Exec(ctx, query, driverID, status)
	if err != nil {
		return fmt.Errorf("failed to update driver status: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no driver found with id: %s", driverID)
	}

	return nil
}

// storeInitialDriverLocation stores the driver's initial location when going online
func (r *repo) storeInitialDriverLocation(ctx context.Context, tx pgx.Tx, driverID string, req models.Location) error {
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
			'Initial online location', $2, $3, true
		)
		RETURNING id
	`

	var coordID string
	err = tx.QueryRow(ctx, insertQuery, driverID, req.Latitude, req.Longitude).Scan(&coordID)
	if err != nil {
		return fmt.Errorf("failed to store driver location: %w", err)
	}

	// Also record in location_history
	const historyQuery = `
		INSERT INTO location_history (
			id, coordinate_id, driver_id, latitude, longitude,
			recorded_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, NOW()
		)
	`

	_, err = tx.Exec(ctx, historyQuery, coordID, driverID, req.Latitude, req.Longitude)
	if err != nil {
		return fmt.Errorf("failed to record location history: %w", err)
	}

	return nil
}

// func (r *repo) CreateSessionDriver(ctx context.Context, data models.Location) (string, error) {
// 	queryInsertDriver := `INSERT INTO driver_sessions(driver_id) VALUES ($1) RETURNING id`
// 	queryUpdateDriver := `UPDATE drivers SET status = $1 WHERE id = $2`
// 	queryInsertLocal := `INSERT INTO location_history(driver_id, latitude, longitude) VALUES ($1, $2, $3)`

// 	tx, err := r.db.Begin(ctx)
// 	if err != nil {
// 		return "", err
// 	}

// 	defer tx.Rollback(ctx)

// 	var id string

// 	err = tx.QueryRow(ctx, queryInsertDriver, data.DriverID).Scan(&id)
// 	if err != nil {
// 		return "", err
// 	}

// 	_, err = tx.Exec(ctx, queryInsertLocal, data.DriverID, data.Latitude, data.Longitude)
// 	if err != nil {
// 		return "", err
// 	}

// 	_, err = tx.Exec(ctx, queryUpdateDriver, models.DriverAvailable, data.DriverID)
// 	if err != nil {
// 		return "", err
// 	}

// 	err = tx.Commit(ctx)
// 	if err != nil {
// 		return "", err
// 	}

// 	return id, nil
// }

func (r *repo) CheckDriverExists(ctx context.Context, driverID string) error {
	query := `SELECT status FROM drivers WHERE id = $1`

	var status string

	err := r.db.QueryRow(ctx, query, driverID).Scan(&status)

	if errors.Is(err, pgx.ErrNoRows) {
		return err // custom error like driver doesn't exist
	} else if err != nil {
		return err // db error
	}

	if status != "OFFLINE" {
		return apperrors.ErrDriverOnline
	}

	return nil
}

func (r *repo) CheckUserExistsAndIsDriver(ctx context.Context, userID string) error {
	query := `SELECT role FROM users WHERE id = $1`

	var role string

	err := r.db.QueryRow(ctx, query, userID).Scan(&role)

	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("user_id does not exist")
	} else if err != nil {
		return err
	}

	if role != "DRIVER" {
		return errors.New("user's role is not driver")
	}

	return nil
}

//--------------------------------------------------------------------------------------------------------------------------------------------

//--------------------------------------------------------------------------------------------------------------------------------------------

//--------------------------------------------------------------------------------------------------------------------------------------------

//--------------------------------------------------------------------------------------------------------------------------------------------

//--------------------------------------------------------------------------------------------------------------------------------------------

//--------------------------------------------------------------------------------------------------------------------------------------------

//--------------------------------------------------------------------------------------------------------------------------------------------

//--------------------------------------------------------------------------------------------------------------------------------------------

func (r *repo) FinishSession(ctx context.Context, driverID string) (*models.FinishDriverResponse, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Check if driver exists
	_, err = r.getDriverForFinish(ctx, tx, driverID)
	if err != nil {
		return nil, err
	}

	// Get active driver session
	activeSession, err := r.getActiveDriverSession(ctx, tx, driverID)
	if err != nil {
		return nil, err
	}

	if activeSession == nil {
		return nil, fmt.Errorf("no active session found for driver")
	}

	// Calculate session summary
	sessionSummary, err := r.calculateSessionSummary(ctx, tx, activeSession, driverID)
	if err != nil {
		return nil, err
	}

	// Update driver status to OFFLINE
	if err := r.updateDriverStatus(ctx, tx, driverID, "OFFLINE"); err != nil {
		return nil, err
	}

	// End the driver session
	if err := r.endDriverSession(ctx, tx, activeSession.ID, sessionSummary); err != nil {
		return nil, err
	}

	// Check if driver has any active rides and handle them
	if err := r.handleActiveRides(ctx, tx, driverID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.FinishDriverResponse{
		Status:         "OFFLINE",
		SessionID:      activeSession.ID,
		SessionSummary: sessionSummary,
		Message:        "You are now offline",
		Timestamp:      time.Now().UTC(),
	}, nil
	// queryUpdateDriver := `UPDATE drivers SET status = $1 WHERE id = $2`
	// queryUpdateSession := `UPDATE driver_sessions SET ended_at = NOW() WHERE id = $1 AND ended_at IS NULL`

	// tx, err := r.db.Begin(ctx)
	// if err != nil {
	// 	return err
	// }

	// defer tx.Rollback(ctx)

	// _, err = tx.Exec(ctx, queryUpdateDriver, models.DriverOffline, id)
	// if err != nil {
	// 	return err
	// }

	// _, err = tx.Exec(ctx, queryUpdateSession, id)
	// if err != nil {
	// 	return err
	// }

	// return tx.Commit(ctx)
}

// getDriverForFinish retrieves and validates driver for going offline
func (r *repo) getDriverForFinish(ctx context.Context, tx pgx.Tx, driverID string) (*models.Driver, error) {
	const query = `
		SELECT d.id, d.license_number, d.vehicle_type, d.status,
		       u.email, u.role, u.status as user_status
		FROM drivers d
		JOIN users u ON d.id = u.id
		WHERE d.id = $1
	`

	var driver models.Driver
	var userEmail, userRole, userStatus string
	err := tx.QueryRow(ctx, query, driverID).Scan(
		&driver.ID, &driver.LicenseNumber, &driver.VehicleType, &driver.Status,
		&userEmail, &userRole, &userStatus,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("driver not found")
		}
		return nil, fmt.Errorf("failed to get driver: %w", err)
	}

	// Validate driver can go offline
	if userStatus != "ACTIVE" {
		return nil, fmt.Errorf("driver account is not active. status: %s", userStatus)
	}

	if userRole != "DRIVER" {
		return nil, fmt.Errorf("user is not a driver")
	}

	return &driver, nil
}

// calculateSessionSummary calculates the summary of the driver's session
func (r *repo) calculateSessionSummary(ctx context.Context, tx pgx.Tx, session *models.DriverSession, driverID string) (*models.DriverSessionSummary, error) {
	// Calculate session duration
	duration := time.Since(session.StartedAt)
	durationHours := duration.Hours()

	// Get rides completed during this session
	ridesCompleted, sessionEarnings, err := r.getSessionRidesStats(ctx, tx, driverID, session.StartedAt)
	if err != nil {
		return nil, err
	}

	// Use session totals if available, otherwise use calculated values
	totalRides := session.TotalRides
	totalEarnings := session.TotalEarnings

	// If session totals are zero, use calculated values
	if totalRides == 0 && ridesCompleted > 0 {
		totalRides = ridesCompleted
	}
	if totalEarnings == 0 && sessionEarnings > 0 {
		totalEarnings = sessionEarnings
	}

	// Get average rating for the session
	averageRating, err := r.getSessionAverageRating(ctx, tx, driverID, session.StartedAt)
	if err != nil {
		// Don't fail if we can't get the rating
		fmt.Println("failed to get session average rating")
		// logger.Warn().Err(err).Msg("failed to get session average rating")
	}

	return &models.DriverSessionSummary{
		DurationHours:  durationHours,
		RidesCompleted: totalRides,
		Earnings:       totalEarnings,
		AverageRating:  averageRating,
	}, nil
}

// getSessionRidesStats gets ride statistics for the session
func (r *repo) getSessionRidesStats(ctx context.Context, tx pgx.Tx, driverID string, sessionStart time.Time) (int, float64, error) {
	const query = `
		SELECT COUNT(*), COALESCE(SUM(final_fare), 0)
		FROM rides 
		WHERE driver_id = $1 
		AND status = 'COMPLETED'
		AND completed_at >= $2
		AND completed_at <= NOW()
	`

	var ridesCompleted int
	var totalEarnings float64

	err := tx.QueryRow(ctx, query, driverID, sessionStart).Scan(&ridesCompleted, &totalEarnings)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get session rides stats: %w", err)
	}

	return ridesCompleted, totalEarnings, nil
}

// getSessionAverageRating gets the average rating for rides in this session
func (r *repo) getSessionAverageRating(ctx context.Context, tx pgx.Tx, driverID string, sessionStart time.Time) (float64, error) {
	const query = `
		SELECT AVG((event_data->>'rating')::numeric)
		FROM ride_events re
		JOIN rides r ON re.ride_id = r.id
		WHERE r.driver_id = $1 
		AND r.status = 'COMPLETED'
		AND r.completed_at >= $2
		AND r.completed_at <= NOW()
		AND re.event_type = 'RIDE_COMPLETED'
		AND event_data ? 'rating'
	`

	var avgRating *float64
	err := tx.QueryRow(ctx, query, driverID, sessionStart).Scan(&avgRating)
	if err != nil {
		return 0, fmt.Errorf("failed to get session average rating: %w", err)
	}

	if avgRating == nil {
		return 0, nil
	}

	return *avgRating, nil
}

// endDriverSession marks the driver session as ended
func (r *repo) endDriverSession(ctx context.Context, tx pgx.Tx, sessionID string, summary *models.DriverSessionSummary) error {
	const query = `
		UPDATE driver_sessions 
		SET ended_at = NOW(),
		    total_rides = $2,
		    total_earnings = $3
		WHERE id = $1
	`

	_, err := tx.Exec(ctx, query, sessionID, summary.RidesCompleted, summary.Earnings)
	if err != nil {
		return fmt.Errorf("failed to end driver session: %w", err)
	}

	return nil
}

// handleActiveRides checks if driver has any active rides and handles them appropriately
func (r *repo) handleActiveRides(ctx context.Context, tx pgx.Tx, driverID string) error {
	const getActiveRidesQuery = `
		SELECT id, status, passenger_id
		FROM rides 
		WHERE driver_id = $1 
		AND status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
	`

	rows, err := tx.Query(ctx, getActiveRidesQuery, driverID)
	if err != nil {
		return fmt.Errorf("failed to get active rides: %w", err)
	}
	defer rows.Close()

	var activeRides []struct {
		ID          string
		Status      string
		PassengerID string
	}

	for rows.Next() {
		var ride struct {
			ID          string
			Status      string
			PassengerID string
		}
		if err := rows.Scan(&ride.ID, &ride.Status, &ride.PassengerID); err != nil {
			return fmt.Errorf("failed to scan active ride: %w", err)
		}
		activeRides = append(activeRides, ride)
	}

	// Handle each active ride
	for _, ride := range activeRides {
		if err := r.handleSingleActiveRide(ctx, tx, ride); err != nil {
			// logger.Error().Err(err).Str("ride_id", ride.ID).Msg("failed to handle active ride")
			fmt.Println("failed to handle active ride")
			// Continue with other rides even if one fails
		}
	}

	return nil
}

// handleSingleActiveRide handles an individual active ride when driver goes offline
func (r *repo) handleSingleActiveRide(ctx context.Context, tx pgx.Tx, ride struct {
	ID          string
	Status      string
	PassengerID string
},
) error {
	// Cancel the ride and notify passenger
	const updateRideQuery = `
		UPDATE rides 
		SET status = 'CANCELLED',
		    cancelled_at = NOW(),
		    cancellation_reason = 'Driver went offline',
		    driver_id = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := tx.Exec(ctx, updateRideQuery, ride.ID)
	if err != nil {
		return fmt.Errorf("failed to cancel ride %s: %w", ride.ID, err)
	}

	// Record cancellation event
	const insertEventQuery = `
		INSERT INTO ride_events (
			id, created_at, ride_id, event_type, event_data
		) VALUES (
			gen_random_uuid(), NOW(), $1, 'RIDE_CANCELLED', $2
		)
	`

	eventData := map[string]interface{}{
		"reason":          "Driver went offline",
		"cancelled_at":    time.Now().UTC().Format(time.RFC3339),
		"previous_status": ride.Status,
	}

	jsonData, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	_, err = tx.Exec(ctx, insertEventQuery, ride.ID, jsonData)
	if err != nil {
		return fmt.Errorf("failed to record cancellation event: %w", err)
	}

	// logger.Info().
	// 	Str("ride_id", ride.ID).
	// 	Str("previous_status", ride.Status).
	// 	Msg("cancelled active ride due to driver going offline")

	return nil
}

// updateFinalDriverLocation updates the driver's final location before going offline
func (r *repo) updateFinalDriverLocation(ctx context.Context, driverID string) error {
	// This would typically get the current location from the request
	// or from the most recent location update
	// For now, we'll just mark the current location with an offline status
	const query = `
		UPDATE coordinates 
		SET address = 'Last online location',
		    updated_at = NOW()
		WHERE entity_id = $1 AND entity_type = 'driver' AND is_current = true
	`

	result, err := r.db.Exec(ctx, query, driverID)
	if err != nil {
		return fmt.Errorf("failed to update final driver location: %w", err)
	}

	if result.RowsAffected() == 0 {
		// logger.Warn().Msg("no current location found to update for driver")
		fmt.Println("no current location found to update for driver")
	}

	return nil
}
