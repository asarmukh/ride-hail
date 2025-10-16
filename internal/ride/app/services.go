package app

import (
	"context"
	"encoding/json"
	"fmt"
	"ride-hail/internal/ride/domain"
	"ride-hail/internal/shared/util"
	"time"
)

type Publisher interface {
	Publish(ctx context.Context, exchange, routingKey string, body []byte) error
	PublishRideStatus(event domain.RideStatusEvent) error
}

type RideService struct {
	repo domain.RideRepository
	pub  Publisher
}

func NewRideService(repo domain.RideRepository, pub Publisher) *RideService {
	return &RideService{repo: repo, pub: pub}
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

func (s *RideService) CreateRide(ctx context.Context, passengerID string, input domain.CreateRideInput) (*domain.Ride, error) {
	if input.PickupLat < -90 || input.PickupLat > 90 || input.PickupLng < -180 || input.PickupLng > 180 {
		return nil, domain.ErrInvalidCoordinates
	}

	if input.DropoffLat < -90 || input.DropoffLat > 90 || input.DropoffLng < -180 || input.DropoffLng > 180 {
		return nil, domain.ErrInvalidCoordinates
	}

	rate, ok := fareRates[input.RideType]
	if !ok {
		return nil, domain.ErrInvalidRideType
	}

	distanceKm := util.Haversine(input.PickupLat, input.PickupLng, input.DropoffLat, input.DropoffLng)
	estimatedDuration := int(distanceKm * 2)
	if estimatedDuration < 1 {
		estimatedDuration = 1
	}

	estimatedFare := rate.Base + (distanceKm * rate.PerKm) + (float64(estimatedDuration) * rate.PerMin)

	rideID := util.GenerateUUID()
	rideNumber := fmt.Sprintf("RIDE_%s_%06d", time.Now().Format("20060102"), time.Now().Unix()%1000000)

	ride := domain.Ride{
		ID:                rideID,
		Number:            rideNumber,
		PassengerID:       passengerID,
		PickupAddress:     input.PickupAddress,
		DropoffAddress:    input.DropoffAddress,
		Status:            "REQUESTED",
		RideType:          input.RideType,
		EstimatedFare:     estimatedFare,
		EstimatedDistance: distanceKm,
		EstimatedDuration: estimatedDuration,
		CreatedAt:         time.Now(),
	}

	if err := s.repo.CreateRide(ctx, ride); err != nil {
		return nil, err
	}

	event := map[string]interface{}{
		"ride_id":            ride.ID,
		"ride_number":        ride.Number,
		"status":             ride.Status,
		"ride_type":          ride.RideType,
		"estimated_fare":     ride.EstimatedFare,
		"estimated_distance": ride.EstimatedDistance,
		"estimated_duration": ride.EstimatedDuration,
		"timestamp":          time.Now().UTC(),
	}
	body, _ := json.Marshal(event)
	routingKey := fmt.Sprintf("ride.request.%s", ride.RideType)
	if err := s.pub.Publish(ctx, "ride_topic", routingKey, body); err != nil {
	}

	return &ride, nil
}

func (s *RideService) CancelRide(ctx context.Context, rideID, passengerID, reason string) (int, error) {
	ride, err := s.repo.GetRideByID(ctx, rideID)
	if err != nil {
		return 0, domain.ErrNotFound
	}

	if ride.PassengerID != passengerID {
		return 0, domain.ErrForbidden
	}

	if ride.Status != "REQUESTED" && ride.Status != "MATCHED" {
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
		return 0, fmt.Errorf("failed to update ride: %w", err)
	}

	err = s.repo.CreateEvent(ctx, rideID, "RIDE_CANCELLED", reason)
	if err != nil {
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
		// s.logger.Warn(fmt.Sprintf("failed to publish cancellation event: %v", err))
	}

	// s.logger.Info(fmt.Sprintf("Ride %s cancelled by passenger %s (refund=%d%%)", rideID, passengerID, refundPercent))

	return refundPercent, nil
}
