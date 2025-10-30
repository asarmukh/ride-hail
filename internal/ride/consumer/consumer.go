package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-hail/internal/ride/app"

	amqp "github.com/rabbitmq/amqp091-go"
)

type DriverResponseConsumer struct {
	service   *app.RideService
	channel   *amqp.Channel
	queue     string
	wsManager WSManager
}

// WSManager interface for sending WebSocket messages
type WSManager interface {
	SendToPassenger(passengerID string, message interface{}) error
}

func NewDriverResponseConsumer(service *app.RideService, ch *amqp.Channel, wsManager WSManager) *DriverResponseConsumer {
	return &DriverResponseConsumer{
		service:   service,
		channel:   ch,
		queue:     "driver_responses",
		wsManager: wsManager,
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
		RideID   string `json:"ride_id"`
		DriverID string `json:"driver_id"`
		Accepted bool   `json:"accepted"`
	}

	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("[driver_responses] invalid JSON: %v", err)
		// Don't requeue malformed messages
		msg.Nack(false, false)
		return
	}

	fmt.Println(payload)

	var err error
	if payload.Accepted {
		log.Printf("[driver_responses] Driver %s accepted ride %s", payload.DriverID, payload.RideID)
		err = c.service.HandleDriverAcceptance(ctx, payload.RideID, payload.DriverID)

		// Send WebSocket notification to passenger on successful acceptance
		if err == nil && c.wsManager != nil {
			ride, rideErr := c.service.GetRideByID(ctx, payload.RideID)
			if rideErr == nil {
				wsUpdate := map[string]interface{}{
					"type":      "ride_status_update",
					"ride_id":   payload.RideID,
					"status":    "MATCHED",
					"message":   "A driver has been matched to your ride",
					"driver_id": payload.DriverID,
				}
				if sendErr := c.wsManager.SendToPassenger(ride.PassengerID, wsUpdate); sendErr != nil {
					log.Printf("[driver_responses] failed to send WebSocket notification: %v", sendErr)
				}
			}
		}
	} else {
		log.Printf("[driver_responses] Driver %s rejected ride %s", payload.DriverID, payload.RideID)
		err = c.service.HandleDriverRejection(ctx, payload.RideID, payload.DriverID)
	}

	if err != nil {
		log.Printf("[driver_responses] handle failed: %v", err)
		// Requeue for retry on processing errors
		msg.Nack(false, true)
		return
	}

	// Acknowledge successful processing
	msg.Ack(false)
}
