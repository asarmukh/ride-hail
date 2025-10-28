package models

import (
	"time"
)

type DriverStatus string

const (
	DriverOffline   DriverStatus = "OFFLINE"
	DriverAvailable DriverStatus = "AVAILABLE"
	DriverBusy      DriverStatus = "BUSY"
	DriverEnRoute   DriverStatus = "EN_ROUTE"
)

type Driver struct {
	ID            string            `db:"id" json:"id"`
	CreatedAt     time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time         `db:"updated_at" json:"updated_at"`
	LicenseNumber string            `db:"license_number" json:"license_number"`
	VehicleType   string            `db:"vehicle_type" json:"vehicle_type"`
	VehicleAttrs  VehicleAttributes `db:"vehicle_attrs" json:"vehicle_attrs"` // jsonb, можно map[string]interface{}
	Rating        float64           `db:"rating" json:"rating"`
	TotalRides    int               `db:"total_rides" json:"total_rides"`
	TotalEarnings float64           `db:"total_earnings" json:"total_earnings"`
	Status        DriverStatus      `db:"status" json:"status"`
	IsVerified    bool              `db:"is_verified" json:"is_verified"`
}

type VehicleAttributes struct {
	Color string `json:"color"`
	Model string `json:"model"`
	Year  int    `json:"year"`
}
