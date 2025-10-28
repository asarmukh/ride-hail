package models

import "time"

const (
	Entity = "driver"
)

type Location struct {
	ID       string `db:"id" json:"id"`
	DriverID string `db:"driver_id" json:"driver_id"`
	// StartedAt     time.Time  `db:"started_at" json:"started_at"`
	Latitude  float64 `db:"latitude" json:"latitude"`
	Longitude float64 `db:"longitude" json:"longitude"`
	// EndedAt       *time.Time `db:"ended_at" json:"ended_at,omitempty"`
	// TotalRides    int        `db:"total_rides" json:"total_rides"`
	// TotalEarnings float64    `db:"total_earnings" json:"total_earnings"`
}

type LocalHistory struct {
	ID             string    `db:"id" json:"id"`
	CoordinateID   string    `db:"coordinate_id" json:"coordinate_id"`
	DriverID       string    `db:"driver_id" json:"driver_id"`
	Latitude       float64   `db:"latitude" json:"latitude"`
	Longitude      float64   `db:"longitude" json:"longitude"`
	AccuracyMeters float64   `db:"accuracy_meters" json:"accuracy_meters"`
	SpeedKmh       float64   `db:"speed_kmh" json:"speed_kmh"`
	HeadingDegrees float64   `db:"heading_degrees" json:"heading_degrees"`
	RecordedAt     time.Time `db:"recorded_at" json:"recorded_at"`
	RideID         string    `db:"ride_id" json:"ride_id"`
}

type Coordinate struct {
	CoordinateID string    `db:"coordinate_id" json:"coordinate_id"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type StartRideReq struct {
	ID             string   `json:"ride_id"`
	DriverLocation Location `json:"driver_location"`
}
