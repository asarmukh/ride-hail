package psql

import (
	"context"
	"errors"
	"fmt"

	"ride-hail/internal/driver/models"

	"github.com/jackc/pgx/v5"
)

// func (r *repo) CreateSessionDriver(ctx context.Context, data models.Location) (string, error) {
// 	queryInsertDriver := `INSERT INTO driver_sessions(driver_id) VALUES ($1) RETURNING id`
// 	queryUpdateDriver := `UPDATE drivers SET status = $1 WHERE id = $2`
// 	queryInsertLocal := `INSERT INTO location_history(driver_id, latitude, longitude, location) VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($3, $2), 4326))`

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

// func (r *repo) CheckDriverExists(ctx context.Context, driverID string) error {
// 	query := `SELECT status FROM drivers WHERE id = $1`

// 	var status string

// 	err := r.db.QueryRow(ctx, query, driverID).Scan(&status)

// 	if errors.Is(err, pgx.ErrNoRows) {
// 		return err // custom error like driver doesn't exist
// 	} else if err != nil {
// 		return err // db error
// 	}

// 	if status != "OFFLINE" {
// 		return apperrors.ErrDriverOnline
// 	}

// 	return nil
// }

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

func (r *repo) FinishSession(ctx context.Context, id string) error {
	queryUpdateDriver := `UPDATE drivers SET status = $1 WHERE id = $2`
	queryUpdateSession := `UPDATE driver_sessions SET ended_at = NOW() WHERE id = $1 AND ended_at IS NULL`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, queryUpdateDriver, models.DriverOffline, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, queryUpdateSession, id)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

//query := `
//		INSERT INTO coordinates (latitude, longitude, location)
//		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($2, $1), 4326))
//		RETURNING id
//`
