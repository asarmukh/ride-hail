package repo

import (
	"context"
	"encoding/json"
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
        entity_id, entity_type, address, latitude, longitude, location
    ) VALUES (
        $1, $2, $3, $4, $5,
        ST_SetSRID(ST_MakePoint($5::double precision, $4::double precision), 4326)
    )
    `,
		ride.ID, "passenger", ride.PickupAddress, ride.PickupLat, ride.PickupLng,
	)
	if err != nil {
		return fmt.Errorf("insert pickup coord failed: %w", err)
	}

	_, err = tx.Exec(ctx, `
    INSERT INTO coordinates (
        entity_id, entity_type, address, latitude, longitude, location
    ) VALUES (
        $1, $2, $3, $4, $5,
        ST_SetSRID(ST_MakePoint($5::double precision, $4::double precision), 4326)
    )
    `,
		ride.ID, "passenger", ride.DropoffAddress, ride.DropoffLat, ride.DropoffLng,
	)
	if err != nil {
		return fmt.Errorf("insert destination coord failed: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *RideRepo) UpdateRideStatus(ctx context.Context, rideID, status, driverID string) error {
	query := `
		UPDATE rides
		SET status = $1,
		    driver_id = CASE WHEN $1 = 'MATCHED' THEN $3 ELSE driver_id END,
		    matched_at = CASE WHEN $1 = 'MATCHED' THEN NOW() ELSE matched_at END,
		    updated_at = NOW()
		WHERE id = $2
	`
	_, err := r.db.Exec(ctx, query, status, rideID, driverID)
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

func (r *RideRepo) UpdateStatus(ctx context.Context, id, status, reason string) error {
	query := `
		UPDATE rides
		SET status = $1,
		    cancellation_reason = $2,
		    cancelled_at = NOW()
		WHERE id = $3
	`
	cmd, err := r.db.Exec(ctx, query, status, reason, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *RideRepo) GetRideStatus(ctx context.Context, rideID string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `
		SELECT status FROM rides WHERE id = $1
	`, rideID).Scan(&status)
	if err != nil {
		return "", err
	}
	return status, nil
}

func (r *RideRepo) CreateEvent(ctx context.Context, rideID, eventType string, payload interface{}) error {
	query := `
		INSERT INTO ride_events (ride_id, event_type, event_data, created_at)
		VALUES ($1, $2, $3::jsonb, NOW())
	`

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	_, err = r.db.Exec(ctx, query, rideID, eventType, string(data))
	if err != nil {
		return fmt.Errorf("failed to insert ride_event: %w", err)
	}

	return nil
}

func (r *RideRepo) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
        SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)
    `, id).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
