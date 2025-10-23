package domain

import "time"

type Ride struct {
	ID                string    `json:"ride_id"`
	Number            string    `json:"ride_number"`
	PassengerID       string    `json:"passenger_id"`
	DriverID          *string   `json:"driver_id,omitempty"`
	PickupAddress     string    `json:"pickup_address"`
	DropoffAddress    string    `json:"destination_address"`
	PickupLat         float64   `json:"pickup_latitude"`
	PickupLng         float64   `json:"pickup_longitude"`
	DropoffLat        float64   `json:"destination_latitude"`
	DropoffLng        float64   `json:"destination_longitude"`
	Status            string    `json:"status"`
	RideType          string    `json:"ride_type"`
	EstimatedFare     float64   `json:"estimated_fare"`
	EstimatedDistance float64   `json:"estimated_distance_km"`
	EstimatedDuration int       `json:"estimated_duration_minutes"`
	CreatedAt         time.Time `json:"created_at"`
}

type CreateRideRequest struct {
	PickupLat      float64 `json:"pickup_latitude"`
	PickupLng      float64 `json:"pickup_longitude"`
	PickupAddress  string  `json:"pickup_address"`
	DropoffLat     float64 `json:"destination_latitude"`
	DropoffLng     float64 `json:"destination_longitude"`
	DropoffAddress string  `json:"destination_address"`
	RideType       string  `json:"ride_type"`
}

type CreateRideResponse struct {
	RideID                string  `json:"ride_id"`
	RideNumber            string  `json:"ride_number"`
	Status                string  `json:"status"`
	EstimatedFare         float64 `json:"estimated_fare"`
	EstimatedDurationMins int     `json:"estimated_duration_minutes"`
	EstimatedDistanceKm   float64 `json:"estimated_distance_km"`
}

type RideStatusEvent struct {
	RideID    string    `json:"ride_id"`
	Status    string    `json:"status"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Phone    string `json:"phone"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
