package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ride-hail/internal/driver/models"
	"ride-hail/internal/ride/api"
	"ride-hail/internal/ride/app"
	"ride-hail/internal/ride/domain"

	amqp "github.com/rabbitmq/amqp091-go"
)

type DriverResponseConsumer struct {
	service *app.RideService
	channel *amqp.Channel
	queue   string
}

func NewDriverResponseConsumer(service *app.RideService, ch *amqp.Channel) *DriverResponseConsumer {
	return &DriverResponseConsumer{
		service: service,
		channel: ch,
		queue:   "driver_responses",
	}
}

func (c *DriverResponseConsumer) Start(ctx context.Context) error {
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
			c.handleDriverResponse(ctx, msg)
		}
	}()
	log.Println("driver_responses consumer started")
	return nil
}

func (c *DriverResponseConsumer) handleDriverResponse(ctx context.Context, msg amqp.Delivery) {
	var payload struct {
		Accepted         bool              `json:"accepted"`
		CorrelationID    string            `json:"correlation_id"`
		DistanceKm       float64           `json:"distance_km"`
		DriverID         string            `json:"driver_id"`
		DriverInfo       domain.DriverInfo `json:"driver_info"`
		DriverLocation   models.Location   `json:"driver_location"`
		EstimatedArrival time.Time         `json:"estimated_arrival"`
		RideID           string            `json:"ride_id"`
	}

	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("[driver_responses] invalid JSON: %v", err)
		// Don't requeue malformed messages
		msg.Nack(false, false)
		return
	}

	fmt.Println("RAW BODY:", string(msg.Body))

	var err error
	if payload.Accepted {
		log.Printf("[driver_responses] Driver %s accepted ride %s", payload.DriverID, payload.RideID)
		err = c.service.HandleDriverAcceptance(ctx, payload.RideID, payload.DriverID)

		// Send WebSocket notification to passenger on successful acceptance
		if err == nil && api.GetGlobalWSManager() != nil {
			ride, rideErr := c.service.GetRideByID(ctx, payload.RideID)
			if rideErr == nil {
				wsUpdate := map[string]interface{}{
					"type":        "ride_status_update",
					"ride_id":     payload.RideID,
					"status":      "MATCHED",
					"message":     "A driver has been matched to your ride",
					"driver_info": map[string]interface{}{
						"driver_id": payload.DriverID,
						"name":      payload.DriverInfo.Name, // replace with dynamic field
						"rating":    payload.DriverInfo.Rating,
						"vehicle": map[string]interface{}{
							"model": payload.DriverInfo.Vehicle.Model,
							"color": payload.DriverInfo.Vehicle.Color,
							"plate": payload.DriverInfo.Vehicle.Plate,
						},
					},
				}
				if sendErr := api.GetGlobalWSManager().SendToPassenger(ride.PassengerID, wsUpdate); sendErr != nil {
					log.Printf("[driver_responses] failed to send WebSocket notification: %v", sendErr)
				}
			}
		}
	} else {
		log.Printf("[driver_responses] Driver %s rejected ride %s", payload.DriverID, payload.RideID)
		err = c.service.HandleDriverRejection(ctx, payload.RideID, payload.DriverID)
	}

	if err != nil {
		log.Printf("[driver_responses] driver response handle failed: %v", err)
		// Requeue for retry on processing errors
		msg.Nack(false, true)
		return
	}

	msg.Ack(false)
}
