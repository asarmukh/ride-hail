package psql

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"ride-hail/internal/driver/models"
)

func (r *repo) UpdateCurrLocation(ctx context.Context, data *models.LocalHistory, update bool) (*models.Coordinate, error) {
	insertCoordinates := `INSERT INTO coordinates(entity_id, entity_type, address, latitude, longitude, fare_amount, distance_km, duration_minutes)
		 VALUES ($1, $2, 'Unknown', $3, $4, 0, 0, 0)
		 RETURNING id, updated_at;`
	updateCoordinates := `UPDATE coordinates SET latitude = $1 AND longitude = $2 WHERE entity_id = $3 RETURNING id, updated_at`
	insertLocalHist := `INSERT INTO location_history(coordinate_id, driver_id, latitude, longitude, accuracy_meters, speed_kmh, heading_degrees)
		 VALUES ($1, $2, $3, $4, $5, $6, $7);`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx)

	result := &models.Coordinate{}

	if !update {
		err := r.db.QueryRow(ctx, insertCoordinates, data.DriverID, "driver", data.Latitude, data.Longitude).Scan(&result.CoordinateID, result.UpdatedAt)
		if err != nil {
			return nil, err
		}

		_, err = r.db.Exec(ctx, insertLocalHist, result.CoordinateID, data.DriverID, data.Latitude, data.Longitude, data.AccuracyMeters, data.SpeedKmh, data.HeadingDegrees)
		if err != nil {
			return nil, err
		}
	} else {
		err := r.db.QueryRow(ctx, updateCoordinates, data.Latitude, data.Longitude, data.DriverID).Scan(&result.CoordinateID, &result.UpdatedAt)
		if err != nil {
			return nil, err
		}

		_, err = r.db.Exec(ctx, insertLocalHist, result.CoordinateID, data.DriverID, data.Latitude, data.Longitude, data.AccuracyMeters, data.SpeedKmh, data.HeadingDegrees)
		if err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *repo) CheckLocationExists(ctx context.Context, driverID string) error {
	query := `SELECT status FROM coordinates WHERE entity_id = $1`

	_, err := r.db.Exec(ctx, query, driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}
