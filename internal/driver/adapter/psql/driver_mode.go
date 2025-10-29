package psql

import (
	"context"
	"errors"
	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/apperrors"

	"github.com/jackc/pgx/v5"
)

func (r *repo) CreateSessionDriver(ctx context.Context, data models.Location) (string, error) {
	queryInsertDriver := `INSERT INTO driver_sessions(driver_id) VALUES ($1) RETURNING id`
	queryUpdateDriver := `UPDATE drivers SET status = $1 WHERE id = $2`
	queryInsertLocal := `INSERT INTO location_history(driver_id, latitude, longitude) VALUES ($1, $2, $3)`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}

	defer tx.Rollback(ctx)

	var id string

	err = tx.QueryRow(ctx, queryInsertDriver, data.DriverID).Scan(&id)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, queryInsertLocal, data.DriverID, data.Latitude, data.Longitude)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, queryUpdateDriver, models.DriverAvailable, data.DriverID)
	if err != nil {
		return "", err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return "", err
	}

	return id, nil
}

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
