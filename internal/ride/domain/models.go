package domain

import "time"

type Ride struct {
    ID               string
    Number           string
    PassengerID      string
    DriverID         *string
    PickupAddress    string
    DropoffAddress   string
    Status           string
    RideType         string
    EstimatedFare    float64
    EstimatedDistance float64
    EstimatedDuration int
    CreatedAt        time.Time
}

type CreateRideInput struct {
    PassengerID      string
    PickupLat        float64
    PickupLng        float64
    PickupAddress    string
    DropoffLat       float64
    DropoffLng       float64
    DropoffAddress   string
    RideType         string
}
