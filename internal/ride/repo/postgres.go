package repo

import (
	"context"
	"fmt"
	"ride-hail/internal/ride/domain"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RideRepo struct {
	db *pgxpool.Pool
}

func NewRideRepo(db *pgxpool.Pool) *RideRepo {
	return &RideRepo{db: db}
}

func (r *RideRepo) CreateRide(ctx context.Context, ride domain.Ride) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
	        INSERT INTO rides (id, passenger_id, ride_number, status, vehicle_type, estimated_fare, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		ride.ID, ride.PassengerID, ride.Number, ride.Status, ride.RideType, ride.EstimatedFare, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert ride failed: %w", err)
	}

	_, err = tx.Exec(ctx, `
     INSERT INTO coordinates (
    id, entity_id, entity_type, address, latitude, longitude, location, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    ST_SetSRID(ST_MakePoint($6::double precision, $5::double precision), 4326),
    NOW()
    `,
		ride.ID, ride.PickupAddress, ride.PickupLat, ride.PickupLng,
	)
	if err != nil {
		return fmt.Errorf("insert pickup coord failed: %w", err)
	}

	_, err = tx.Exec(ctx, `
        INSERT INTO coordinates (id, entity_id, entity_type, address, latitude, longitude, location, created_at)
        VALUES ( $1, 'destination', $2, $3, $4, $5,
                ST_SetSRID(ST_MakePoint($4, $3), 4326), NOW())
    `,
		ride.ID, ride.DropoffAddress, ride.DropoffLat, ride.DropoffLng,
	)
	if err != nil {
		return fmt.Errorf("insert destination coord failed: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *RideRepo) UpdateRideStatus(ctx context.Context, rideID string, status string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE rides SET status = $1 WHERE id = $2
	`, status, rideID)
	return err
}

func (r *RideRepo) GetRideByID(ctx context.Context, rideID string) (*domain.Ride, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, passenger_id, ride_number, status, vehicle_type, estimated_fare, created_at
		FROM rides
		WHERE id = $1
	`, rideID)

	var ride domain.Ride
	err := row.Scan(&ride.ID, &ride.PassengerID, &ride.Number, &ride.Status, &ride.RideType, &ride.EstimatedFare, &ride.CreatedAt)
	if err != nil {
		return nil, err
	}

	row = r.db.QueryRow(ctx, `
		SELECT address, latitude, longitude
		FROM coordinates
		WHERE entity_id = $1 AND entity_type = 'pickup'
		LIMIT 1
	`, rideID)
	row.Scan(&ride.PickupAddress, &ride.PickupLat, &ride.PickupLng)

	row = r.db.QueryRow(ctx, `
		SELECT address, latitude, longitude
		FROM coordinates
		WHERE entity_id = $1 AND entity_type = 'destination'
		LIMIT 1
	`, rideID)
	row.Scan(&ride.DropoffAddress, &ride.DropoffLat, &ride.DropoffLng)

	return &ride, nil
}
