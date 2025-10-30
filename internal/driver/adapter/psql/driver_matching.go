package psql

import (
	"context"
	"encoding/json"
	"fmt"
)

func (r *repo) FindNearbyDrivers(ctx context.Context, lat, lng float64, vehicleType string, radiusKm float64) ([]NearbyDriver, error) {
	query := `
		SELECT
			d.id,
			u.email,
			d.rating,
			d.total_rides,
			d.total_rides - COALESCE(
				(SELECT COUNT(*) FROM rides WHERE driver_id = d.id AND status = 'CANCELLED'),
				0
			) as completed_rides,
			d.vehicle_type,
			d.vehicle_attrs,
			c.latitude,
			c.longitude,
			ST_Distance(
				ST_MakePoint(c.longitude, c.latitude)::geography,
				ST_MakePoint($1, $2)::geography
			) / 1000 as distance_km
		FROM drivers d
		JOIN users u ON d.id = u.id
		JOIN coordinates c ON c.entity_id = d.id
			AND c.entity_type = 'driver'
			AND c.is_current = true
		WHERE d.status = 'AVAILABLE'
			AND d.vehicle_type = $3
			AND ST_DWithin(
				ST_MakePoint(c.longitude, c.latitude)::geography,
				ST_MakePoint($1, $2)::geography,
				$4
			)
		ORDER BY distance_km ASC, d.rating DESC
		LIMIT 10
	`

	rows, err := r.db.Query(ctx, query, lng, lat, vehicleType, radiusKm*1000) // Convert km to meters
	if err != nil {
		return nil, fmt.Errorf("failed to query nearby drivers: %w", err)
	}
	defer rows.Close()

	drivers := []NearbyDriver{}
	for rows.Next() {
		var driver NearbyDriver
		var driverUUID string
		var vehicleAttrsJSON []byte

		err := rows.Scan(
			&driverUUID,
			&driver.Email,
			&driver.Rating,
			&driver.TotalRides,
			&driver.CompletedRides,
			&driver.VehicleType,
			&vehicleAttrsJSON,
			&driver.Latitude,
			&driver.Longitude,
			&driver.DistanceKm,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan driver: %w", err)
		}

		driver.ID = driverUUID

		// Parse vehicle attributes JSON
		if len(vehicleAttrsJSON) > 0 {
			var attrs map[string]interface{}
			if err := json.Unmarshal(vehicleAttrsJSON, &attrs); err == nil {
				driver.VehicleAttrs = attrs
			}
		}

		drivers = append(drivers, driver)
	}

	return drivers, nil
}

func (r *repo) UpdateDriverStatus(ctx context.Context, driverID string, status string) error {

	query := `UPDATE drivers SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, driverID)
	if err != nil {
		return fmt.Errorf("failed to update driver status: %w", err)
	}

	return nil
}
