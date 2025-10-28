package models

import "time"

type DriverSession struct {
	ID            string     `db:"id" json:"id"`
	DriverID      string     `db:"driver_id" json:"driver_id"`
	StartedAt     time.Time  `db:"started_at" json:"started_at"`
	EndedAt       *time.Time `db:"ended_at" json:"ended_at,omitempty"`
	TotalRides    int        `db:"total_rides" json:"total_rides"`
	TotalEarnings float64    `db:"total_earnings" json:"total_earnings"`
}
