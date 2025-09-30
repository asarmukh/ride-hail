package domain

import "context"

type RideRepository interface {
    CreateRide(ctx context.Context, ride Ride) error
    UpdateRideStatus(ctx context.Context, rideID string, status string) error
    GetRideByID(ctx context.Context, rideID string) (*Ride, error)
}

type RideService interface {
    CreateRide(ctx context.Context, input CreateRideInput) (*Ride, error)
    CancelRide(ctx context.Context, rideID string) error
}