package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"ride-hail/internal/ride/api"
	"ride-hail/internal/ride/app"
	"ride-hail/internal/shared/util"

	amqp "github.com/rabbitmq/amqp091-go"
)

type LocationConsumer struct {
	service   *app.RideService
	channel   *amqp.Channel
	queue     string
	wsManager *api.WSManager
}

type LocationUpdate struct {
	DriverID       string             `json:"driver_id"`
	RideID         string             `json:"ride_id"`
	Location       LocationCoordinate `json:"location"`
	SpeedKmh       float64            `json:"speed_kmh"`
	HeadingDegrees float64            `json:"heading_degrees"`
	AccuracyMeters float64            `json:"accuracy_meters"`
	Timestamp      string             `json:"timestamp"`
	CoordinateID   string             `json:"coordinate_id"`
}

type LocationCoordinate struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func NewLocationConsumer(service *app.RideService, ch *amqp.Channel, wsManager *api.WSManager) *LocationConsumer {
	return &LocationConsumer{
		service:   service,
		channel:   ch,
		queue:     "location_updates_ride",
		wsManager: wsManager,
	}
}

func (c *LocationConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queue,
		"",
		false, // manual acknowledgment
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			c.handleLocationUpdate(ctx, msg)
		}
	}()
	log.Println("location_updates_ride consumer started")
	return nil
}

func (c *LocationConsumer) handleLocationUpdate(ctx context.Context, msg amqp.Delivery) {
	var update LocationUpdate
	if err := json.Unmarshal(msg.Body, &update); err != nil {
		log.Printf("[location_updates_ride] invalid JSON: %v", err)
		// Don't requeue malformed messages
		msg.Nack(false, false)
		return
	}

	// Skip if no ride_id (driver not on active ride)
	if update.RideID == "" {
		msg.Ack(false)
		return
	}

	// Get ride details
	ride, err := c.service.GetRideByID(ctx, update.RideID)
	if err != nil {
		log.Printf("[location_updates_ride] failed to get ride %s: %v", update.RideID, err)
		msg.Nack(false, true) // Requeue for retry
		return
	}

	// Only process if ride is in EN_ROUTE, MATCHED, or ARRIVED status
	if ride.Status != "EN_ROUTE" && ride.Status != "MATCHED" && ride.Status != "ARRIVED" {
		msg.Ack(false)
		return
	}

	// Calculate ETA if driver is en route
	var eta time.Time
	var distanceToPickup float64

	if ride.Status == "EN_ROUTE" || ride.Status == "MATCHED" {
		// Calculate distance to pickup using Haversine formula
		distanceToPickup = util.Haversine(
			update.Location.Lat, update.Location.Lng,
			ride.PickupLat, ride.PickupLng,
		)

		// Calculate ETA
		eta = calculateETA(distanceToPickup, update.SpeedKmh)
	}

	// Send update to passenger via WebSocket
	if c.wsManager != nil {
		wsUpdate := map[string]interface{}{
			"type":    "driver_location_update",
			"ride_id": ride.ID,
			"driver_location": map[string]float64{
				"lat": update.Location.Lat,
				"lng": update.Location.Lng,
			},
			"speed_kmh":       update.SpeedKmh,
			"heading_degrees": update.HeadingDegrees,
		}

		if ride.Status == "EN_ROUTE" || ride.Status == "MATCHED" {
			wsUpdate["estimated_arrival"] = eta.Format(time.RFC3339)
			wsUpdate["distance_to_pickup_km"] = distanceToPickup
		}

		if err := c.wsManager.SendToPassenger(ride.PassengerID, wsUpdate); err != nil {
			log.Printf("[location_updates_ride] failed to send to passenger: %v", err)
		}
	}

	msg.Ack(false)
}

// calculateETA calculates estimated time of arrival
func calculateETA(distanceKm float64, currentSpeedKmh float64) time.Time {
	// Use current speed if available, otherwise assume 40 km/h average
	speed := currentSpeedKmh
	if speed < 10 { // Too slow or stopped
		speed = 40.0
	}

	durationHours := distanceKm / speed
	durationMinutes := durationHours * 60

	return time.Now().Add(time.Duration(durationMinutes) * time.Minute)
}
