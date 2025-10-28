package domain

import "errors"

var (
	ErrInvalidCoordinates = errors.New("invalid coordinates")
	ErrInvalidRideType    = errors.New("invalid ride type")
	ErrNotFound           = errors.New("ride not found")
	ErrForbidden          = errors.New("forbidden action")
	ErrInvalidStatus      = errors.New("invalid ride status")
	ErrUserNotFound       = errors.New("user not found")
)
