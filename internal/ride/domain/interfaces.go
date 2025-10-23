package domain

import "context"

type RideRepository interface {
	CreateRide(ctx context.Context, ride Ride) error
	UpdateRideStatus(ctx context.Context, rideID string, status string, driverID string) error
	GetRideByID(ctx context.Context, rideID string) (*Ride, error)
	UpdateStatus(ctx context.Context, id string, status, reason string) error
	CreateEvent(ctx context.Context, rideID, eventType string, payload interface{}) error
	Exists(ctx context.Context, id string) (bool, error)
}

type RideService interface {
	CreateRide(ctx context.Context, passengerID string, input CreateRideRequest) (*Ride, error)
	CancelRide(ctx context.Context, rideID, passengerID, reason string) (int, error)
	HandleDriverAcceptance(ctx context.Context, rideID, driverID string) error
	HandleDriverRejection(ctx context.Context, rideID, driverID string) error
}
