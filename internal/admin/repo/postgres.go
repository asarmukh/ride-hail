package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminRepo struct {
	db *pgxpool.Pool
}

func NewAdminRepo(db *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{db: db}
}

type SystemMetrics struct {
	ActiveRides                int     `json:"active_rides"`
	AvailableDrivers           int     `json:"available_drivers"`
	BusyDrivers                int     `json:"busy_drivers"`
	TotalRidesToday            int     `json:"total_rides_today"`
	TotalRevenueToday          float64 `json:"total_revenue_today"`
	AverageWaitTimeMinutes     float64 `json:"average_wait_time_minutes"`
	AverageRideDurationMinutes float64 `json:"average_ride_duration_minutes"`
	CancellationRate           float64 `json:"cancellation_rate"`
}

type DriverDistribution map[string]int

type ActiveRide struct {
	RideID                string     `json:"ride_id"`
	RideNumber            string     `json:"ride_number"`
	Status                string     `json:"status"`
	PassengerID           string     `json:"passenger_id"`
	DriverID              *string    `json:"driver_id"`
	PickupAddress         string     `json:"pickup_address"`
	DestinationAddress    string     `json:"destination_address"`
	StartedAt             *time.Time `json:"started_at"`
	EstimatedCompletion   float32    `json:"estimated_completion"`
	CurrentDriverLocation *Location  `json:"current_driver_location"`
	DistanceCompletedKm   float64    `json:"distance_completed_km"`
	DistanceRemainingKm   float64    `json:"distance_remaining_km"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (r *AdminRepo) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	metrics := &SystemMetrics{}

	// Active rides count
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides
		WHERE status NOT IN ('COMPLETED', 'CANCELLED')
	`).Scan(&metrics.ActiveRides)
	if err != nil {
		return nil, err
	}

	// Available drivers
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM drivers WHERE status = 'AVAILABLE'
	`).Scan(&metrics.AvailableDrivers)
	if err != nil {
		return nil, err
	}

	// Busy drivers
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM drivers WHERE status IN ('BUSY', 'EN_ROUTE')
	`).Scan(&metrics.BusyDrivers)
	if err != nil {
		return nil, err
	}

	// Total rides today
	err = r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides
		WHERE DATE(created_at) = CURRENT_DATE
	`).Scan(&metrics.TotalRidesToday)
	if err != nil {
		return nil, err
	}

	// Total revenue today
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(final_fare), 0) FROM rides
		WHERE status = 'COMPLETED'
		AND DATE(completed_at) = CURRENT_DATE
	`).Scan(&metrics.TotalRevenueToday)
	if err != nil {
		return nil, err
	}

	// Average wait time
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (matched_at - requested_at))/60), 0)
		FROM rides
		WHERE matched_at IS NOT NULL
		AND DATE(requested_at) = CURRENT_DATE
	`).Scan(&metrics.AverageWaitTimeMinutes)
	if err != nil {
		return nil, err
	}

	// Average ride duration
	err = r.db.QueryRow(ctx, `
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))/60), 0)
		FROM rides
		WHERE status = 'COMPLETED'
		AND DATE(completed_at) = CURRENT_DATE
	`).Scan(&metrics.AverageRideDurationMinutes)
	if err != nil {
		return nil, err
	}

	// Cancellation rate
	var totalToday, cancelledToday int
	err = r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'CANCELLED') as cancelled
		FROM rides
		WHERE DATE(created_at) = CURRENT_DATE
	`).Scan(&totalToday, &cancelledToday)
	if err != nil {
		return nil, err
	}

	if totalToday > 0 {
		metrics.CancellationRate = float64(cancelledToday) / float64(totalToday)
	}

	return metrics, nil
}

func (r *AdminRepo) GetDriverDistribution(ctx context.Context) (DriverDistribution, error) {
	distribution := make(DriverDistribution)

	rows, err := r.db.Query(ctx, `
		SELECT vehicle_type, COUNT(*)
		FROM drivers
		WHERE status IN ('AVAILABLE', 'BUSY', 'EN_ROUTE')
		GROUP BY vehicle_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var vehicleType string
		var count int
		if err := rows.Scan(&vehicleType, &count); err != nil {
			return nil, err
		}
		distribution[vehicleType] = count
	}

	return distribution, nil
}

func (r *AdminRepo) GetActiveRides(ctx context.Context, page, pageSize int) ([]ActiveRide, int, error) {
	offset := (page - 1) * pageSize

	// Get total count
	var totalCount int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM rides
		WHERE status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
	`).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Get rides with details
	rows, err := r.db.Query(ctx, `
		SELECT
			r.id, r.ride_number, r.status, r.passenger_id, r.driver_id,
			pc.address as pickup_address, dc.address as destination_address,
			r.started_at, r.estimated_fare,
			lh.latitude as current_lat, lh.longitude as current_lng
		FROM rides r
		LEFT JOIN coordinates pc ON r.pickup_coordinate_id = pc.id
		LEFT JOIN coordinates dc ON r.destination_coordinate_id = dc.id
		LEFT JOIN LATERAL (
			SELECT latitude, longitude
			FROM location_history
			WHERE driver_id = r.driver_id
			ORDER BY recorded_at DESC
			LIMIT 1
		) lh ON true
		WHERE r.status IN ('MATCHED', 'EN_ROUTE', 'ARRIVED', 'IN_PROGRESS')
		ORDER BY r.requested_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rides := []ActiveRide{}
	for rows.Next() {
		var ride ActiveRide
		var currentLat, currentLng *float64

		err := rows.Scan(
			&ride.RideID,
			&ride.RideNumber,
			&ride.Status,
			&ride.PassengerID,
			&ride.DriverID,
			&ride.PickupAddress,
			&ride.DestinationAddress,
			&ride.StartedAt,
			&ride.EstimatedCompletion, // Using estimated_fare as placeholder
			&currentLat,
			&currentLng,
		)
		if err != nil {
			return nil, 0, err
		}

		if currentLat != nil && currentLng != nil {
			ride.CurrentDriverLocation = &Location{
				Latitude:  *currentLat,
				Longitude: *currentLng,
			}
		}

		// TODO: Calculate distance completed and remaining using PostGIS
		ride.DistanceCompletedKm = 0.0
		ride.DistanceRemainingKm = 0.0

		rides = append(rides, ride)
	}

	return rides, totalCount, nil
}
