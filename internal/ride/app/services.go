package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ride-hail/internal/ride/domain"
	"ride-hail/internal/shared/util"
)

type RideService struct {
	repo   domain.RideRepository
	pub    domain.Publisher
	logger *util.Logger
}

func NewRideService(repo domain.RideRepository, pub domain.Publisher, logger *util.Logger) *RideService {
	return &RideService{repo: repo, pub: pub, logger: logger}
}

var fareRates = map[string]struct {
	Base   float64
	PerKm  float64
	PerMin float64
}{
	"ECONOMY": {Base: 500, PerKm: 100, PerMin: 50},
	"PREMIUM": {Base: 800, PerKm: 120, PerMin: 60},
	"XL":      {Base: 1000, PerKm: 150, PerMin: 75},
}

func (s *RideService) CreateRide(ctx context.Context, passengerID string, input domain.CreateRideRequest) (*domain.Ride, error) {
	instance := "RideService.CreateRide"
	start := time.Now()

	if input.PickupLat < -90 || input.PickupLat > 90 || input.PickupLng < -180 || input.PickupLng > 180 {
		s.logger.Warn(instance, fmt.Sprintf("invalid pickup coordinates: lat=%.4f, lng=%.4f", input.PickupLat, input.PickupLng))
		return nil, domain.ErrInvalidCoordinates
	}

	if input.DropoffLat < -90 || input.DropoffLat > 90 || input.DropoffLng < -180 || input.DropoffLng > 180 {
		s.logger.Warn(instance, fmt.Sprintf("invalid dropoff coordinates: lat=%.4f, lng=%.4f", input.DropoffLat, input.DropoffLng))
		return nil, domain.ErrInvalidCoordinates
	}

	rate, ok := fareRates[input.RideType]
	if !ok {
		s.logger.Warn(instance, fmt.Sprintf("invalid ride type: %s", input.RideType))
		return nil, domain.ErrInvalidRideType
	}

	distanceKm := util.Haversine(input.PickupLat, input.PickupLng, input.DropoffLat, input.DropoffLng)
	estimatedDuration := int(distanceKm * 2)
	if estimatedDuration < 1 {
		estimatedDuration = 1
	}

	estimatedFare := rate.Base + (distanceKm * rate.PerKm) + (float64(estimatedDuration) * rate.PerMin)

	rideID, err := util.GenerateUUID()
	if err != nil {
		s.logger.Warn(instance, "failed generating uuid")
		return nil, domain.ErrInternalError
	}
	now := time.Now()
	rideNumber := fmt.Sprintf("RIDE_%s_%s_%03d",
		now.Format("20060102"),        // YYYYMMDD
		now.Format("150405"),          // HHMMSS
		now.Nanosecond()/1000000%1000, // XXX (0-999)
	)

	ride := domain.Ride{
		ID:                rideID,
		Number:            rideNumber,
		PassengerID:       passengerID,
		PickupAddress:     input.PickupAddress,
		PickupLat:         input.PickupLat,
		PickupLng:         input.PickupLng,
		DropoffAddress:    input.DropoffAddress,
		DropoffLat:        input.DropoffLat,
		DropoffLng:        input.PickupLng,
		Status:            "REQUESTED",
		RideType:          input.RideType,
		EstimatedFare:     estimatedFare,
		EstimatedDistance: distanceKm,
		EstimatedDuration: estimatedDuration,
		CreatedAt:         time.Now(),
	}

	if err := s.repo.CreateRide(ctx, ride); err != nil {
		s.logger.Error(instance, "Failed to create ride in database", err)
		return nil, err
	}

	event := map[string]interface{}{
		"ride_id":     ride.ID,
		"ride_number": ride.Number,
		"pickup_location": map[string]interface{}{
			"lat":     input.PickupLat,
			"lng":     input.PickupLng,
			"address": input.PickupAddress,
		},
		"destination_location": map[string]interface{}{
			"lat":     input.DropoffLat,
			"lng":     input.DropoffLng,
			"address": input.DropoffAddress,
		},
		"ride_type":       ride.RideType,
		"estimated_fare":  ride.EstimatedFare,
		"timeout_seconds": 120,
		"correlation_id":  "",
	}
	body, _ := json.Marshal(event)
	routingKey := fmt.Sprintf("ride.request.%s", ride.RideType)
	if err := s.pub.Publish(ctx, "ride_topic", routingKey, body); err != nil {
		s.logger.Warn(instance, fmt.Sprintf("failed to publish ride request: %v", err))
	} else {
		s.logger.OK(instance, fmt.Sprintf("ride request published to %s", routingKey))
	}

	s.logger.Info(instance, fmt.Sprintf("ride created successfully [ride_id=%s, fare=%.2f, type=%s, duration_ms=%d]",
		rideID, estimatedFare, input.RideType, time.Since(start).Milliseconds()))

	go s.startDriverMatchtimer(ctx, rideID, 2*time.Minute)

	return &ride, nil
}

func (s *RideService) CancelRide(ctx context.Context, rideID, passengerID, reason string) (int, error) {
	instance := "RideService.CancelRide"
	start := time.Now()

	ride, err := s.repo.GetRideByID(ctx, rideID)
	if err != nil {
		s.logger.Warn(instance, fmt.Sprintf("ride not found: %s", rideID))
		return 0, domain.ErrNotFound
	}

	if ride.PassengerID != passengerID {
		s.logger.Warn(instance, fmt.Sprintf("unauthorized cancellation attempt by passenger %s for ride %s", passengerID, rideID))
		return 0, domain.ErrForbidden
	}

	if ride.Status != "REQUESTED" && ride.Status != "MATCHED" {
		s.logger.Warn(instance, fmt.Sprintf("invalid status for cancellation: %s", ride.Status))
		return 0, domain.ErrInvalidStatus
	}

	refundPercent := 0
	switch ride.Status {
	case "REQUESTED":
		refundPercent = 100
	case "MATCHED":
		refundPercent = 90
	default:
		refundPercent = 0
	}

	err = s.repo.UpdateStatus(ctx, rideID, "CANCELLED", reason)
	if err != nil {
		s.logger.Error(instance, "Failed to update ride status", err)
		return 0, fmt.Errorf("failed to update ride: %w", err)
	}

	err = s.repo.CreateEvent(ctx, rideID, "RIDE_CANCELLED", reason)
	if err != nil {
		s.logger.Error(instance, "Failed to create cancellation event", err)
		return 0, fmt.Errorf("failed to create event: %w", err)
	}

	event := domain.RideStatusEvent{
		RideID:    rideID,
		Status:    "CANCELLED",
		Reason:    reason,
		Timestamp: time.Now().UTC(),
	}
	err = s.pub.PublishRideStatus(event)
	if err != nil {
		s.logger.Warn(instance, fmt.Sprintf("failed to publish ride cancel event: %v", err))
	}

	s.logger.OK(instance, fmt.Sprintf("ride %s cancelled (refund=%d%%, duration=%dms)", rideID, refundPercent, time.Since(start).Milliseconds()))

	return refundPercent, nil
}

func (s *RideService) HandleDriverAcceptance(ctx context.Context, rideID, driverID string) error {
	instance := "RideService.HandleDriverAcceptance"
	start := time.Now()

	err := s.repo.UpdateRideStatus(ctx, rideID, "MATCHED", driverID)
	if err != nil {
		s.logger.Error(instance, "Failed to update ride status to MATCHED", err)
		return err
	}

	event := map[string]interface{}{
		"ride_id":   rideID,
		"driver_id": driverID,
		"status":    "MATCHED",
		"timestamp": time.Now().UTC(),
	}
	body, _ := json.Marshal(event)

	if err := s.pub.Publish(ctx, "ride_topic", "ride.status.matched", body); err != nil {
		s.logger.Warn(instance, fmt.Sprintf("failed to publish MATCHED event: %v", err))
		log.Printf("publish MATCHED failed: %v", err)
	} else {
		s.logger.OK(instance, fmt.Sprintf("published MATCHED event for ride %s", rideID))
	}

	if err := s.repo.CreateEvent(ctx, rideID, "DRIVER_MATCHED", string(body)); err != nil {
		s.logger.Warn(instance, fmt.Sprintf("failed to record event: %v", err))
	}

	s.logger.Info(instance, fmt.Sprintf("driver %s MATCHED to ride %s (took %dms)", driverID, rideID, time.Since(start).Milliseconds()))
	return nil
}

func (s *RideService) HandleDriverRejection(ctx context.Context, rideID, driverID string) error {
	log.Printf("[ride %s] driver %s rejected ride", rideID, driverID)
	return nil
}

func (s *RideService) GetRideByID(ctx context.Context, rideID string) (*domain.Ride, error) {
	return s.repo.GetRideByID(ctx, rideID)
}

func (s *RideService) startDriverMatchtimer(ctx context.Context, rideID string, duration time.Duration) {
	instance := "RideService.startDriverMatchtimer"

	// Use a channel to listen for the cancellation signal from the context
	timer := time.NewTimer(duration)
	defer timer.Stop()
	fmt.Println("fhaewfhiewfawejfwae------------------------")
	select {
	case <-ctx.Done(): // Context is cancelled
		s.logger.Info(instance, fmt.Sprintf("ride %s timer cancelled", rideID))
		return
	case <-timer.C: // Timer finished successfully
		// Continue with the logic after timer finishes

	}

	// After the timer finishes or the context isn't cancelled, proceed with the status check
	currentStatus, err := s.repo.GetRideStatus(ctx, rideID)
	if err != nil {
		s.logger.Warn(instance, fmt.Sprintf("failed to fetch ride status: %v", err))
		return
	}

	if currentStatus == "REQUESTED" {
		_ = s.repo.UpdateRideStatus(ctx, rideID, "CANCELLED", "")
		_ = s.repo.CreateEvent(ctx, rideID, "RIDE_CANCELLED", `{"reason": "No drivers available"}`)
		s.logger.Info(instance, fmt.Sprintf("ride %s auto-cancelled after %.0fs (no drivers matched)", rideID, duration.Seconds()))
	}
}

// UpdateRideStartTime updates the started_at timestamp for a ride
func (s *RideService) UpdateRideStartTime(ctx context.Context, rideID, startedAt string) error {
	// This would need a repo method to update started_at field
	// For now, just log it
	s.logger.Info("RideService.UpdateRideStartTime", fmt.Sprintf("ride %s started at %s", rideID, startedAt))
	return nil
}

// UpdateRideCompletion updates completion details for a ride
func (s *RideService) UpdateRideCompletion(ctx context.Context, rideID, completedAt string, finalFare, actualDistanceKm float64, actualDurationMin int) error {
	// This would need a repo method to update completion fields
	// For now, just log it
	s.logger.Info("RideService.UpdateRideCompletion", fmt.Sprintf("ride %s completed: fare=%.2f, distance=%.2fkm, duration=%dmin", rideID, finalFare, actualDistanceKm, actualDurationMin))
	return nil
}

// UpdateRideStatus updates the status of a ride
func (s *RideService) UpdateRideStatus(ctx context.Context, rideID, status, driverID string) error {
	return s.repo.UpdateRideStatus(ctx, rideID, status, driverID)
}

// RecordEvent records an event in the ride_events table
func (s *RideService) RecordEvent(ctx context.Context, rideID, eventType string, eventData map[string]interface{}) error {
	return s.repo.CreateEvent(ctx, rideID, eventType, eventData)
}
