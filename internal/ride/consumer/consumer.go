package consumer

import (
	"context"
	"encoding/json"
	"log"
	"ride-hail/internal/ride/app"

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
		true,
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
			var payload struct {
				RideID   string `json:"ride_id"`
				DriverID string `json:"driver_id"`
				Accepted bool   `json:"accepted"`
			}

			if err := json.Unmarshal(msg.Body, &payload); err != nil {
				log.Printf("[driver_responses] invalid JSON: %v", err)
				continue
			}

			if payload.Accepted {
				log.Printf("[driver_responses] Driver %s accepted ride %s", payload.DriverID, payload.RideID)
				if err := c.service.HandleDriverAcceptance(ctx, payload.RideID, payload.DriverID); err != nil {
					log.Printf("[driver_responses] handle acceptance failed: %v", err)
				}
			} else {
				log.Printf("[driver_responses] Driver %s rejected ride %s", payload.DriverID, payload.RideID)
				if err := c.service.HandleDriverRejection(ctx, payload.RideID, payload.DriverID); err != nil {
					log.Printf("[driver_responses] handle rejection failed: %v", err)
				}
			}
		}
	}()
	log.Println("driver_responses consumer started")
	return nil
}
