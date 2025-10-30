package psql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ride-hail/internal/driver/models"

	"github.com/jackc/pgx/v5"
)

func (r *repo) InsertDriver(ctx context.Context, driverData *models.Driver) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert struct → JSONB for Postgres
	vehicleJSON, err := json.Marshal((*driverData).VehicleAttrs)
	if err != nil {
		return fmt.Errorf("failed to marshal vehicle_attrs: %w", err)
	}

	query := `
		INSERT INTO drivers (
			id,
			license_number,
			vehicle_type,
			vehicle_attrs,
			status,
			is_verified
		) VALUES (
			$1, $2, $3, $4, 'OFFLINE', $5
		)
		RETURNING created_at, updated_at;
	`

	// Send the query
	err = r.db.QueryRow(ctx, query,
		(*driverData).ID,
		(*driverData).LicenseNumber,
		(*driverData).VehicleType,
		vehicleJSON,
		(*driverData).IsVerified,
	).Scan(&(*driverData).CreatedAt, &(*driverData).UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert driver failed: %w", err)
	}

	return nil
}

func (r *repo) CheckLicenseNumberExists(ctx context.Context, licenseNumber string) (bool, error) {
	query := `SELECT 1 FROM drivers WHERE license_number = $1 LIMIT 1`

	var exists int
	err := r.db.QueryRow(ctx, query, licenseNumber).Scan(&exists)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
