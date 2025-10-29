package psql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ride-hail/internal/driver/models"
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
			status
		) VALUES (
			$1, $2, $3, $4, $5
		)
		RETURNING created_at, updated_at;
	`

	// Send the query
	err = r.db.QueryRow(ctx, query,
		(*driverData).ID,
		(*driverData).LicenseNumber,
		(*driverData).VehicleType,
		vehicleJSON,
		(*driverData).Status,
	).Scan(&(*driverData).CreatedAt, &(*driverData).UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert driver failed: %w", err)
	}

	return nil
}
