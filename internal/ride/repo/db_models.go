package repo

import "time"

type RideRow struct {
    ID                   string    `db:"id"`
    RideNumber           string    `db:"ride_number"`
    PassengerID          string    `db:"passenger_id"`
    DriverID             *string   `db:"driver_id"`
    Status               string    `db:"status"`
    VehicleType          string    `db:"vehicle_type"`
    EstimatedFare        float64   `db:"estimated_fare"`
    PickupCoordinateID   *string   `db:"pickup_coordinate_id"`
    DestinationCoordinateID *string `db:"destination_coordinate_id"`
    CreatedAt            time.Time `db:"created_at"`
}

type CoordinateRow struct {
    ID        string    `db:"id"`
    EntityID  string    `db:"entity_id"`
    EntityType string   `db:"entity_type"`
    Address   string    `db:"address"`
    Latitude  float64   `db:"latitude"`
    Longitude float64   `db:"longitude"`
    IsCurrent bool      `db:"is_current"`
    CreatedAt time.Time `db:"created_at"`
}