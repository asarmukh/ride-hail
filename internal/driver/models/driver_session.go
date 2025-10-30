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

type FinishDriverResponse struct {
	Status         string                `json:"status"`
	SessionID      string                `json:"session_id"`
	SessionSummary *DriverSessionSummary `json:"session_summary,omitempty"`
	Message        string                `json:"message"`
	Timestamp      time.Time             `json:"timestamp"`
}

type DriverSessionSummary struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted int     `json:"rides_completed"`
	Earnings       float64 `json:"earnings"`
	AverageRating  float64 `json:"average_rating,omitempty"`
}
