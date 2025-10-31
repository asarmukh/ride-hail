package consumer

import (
	"context"
	"encoding/json"
	"log"
	"ride-hail/internal/ride/api"
	"ride-hail/internal/ride/app"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RideStatusConsumer struct {
	service   *app.RideService
	channel   *amqp.Channel
	queue     string
	wsManager *api.WSManager
}

type RideStatusUpdate struct {
	RideID            string             `json:"ride_id"`
	DriverID          string             `json:"driver_id"`
	Status            string             `json:"status"`
	StartedAt         string             `json:"started_at,omitempty"`
	CompletedAt       string             `json:"completed_at,omitempty"`
	FinalFare         float64            `json:"final_fare,omitempty"`
	ActualDistanceKm  float64            `json:"actual_distance_km,omitempty"`
	ActualDurationMin int                `json:"actual_duration_min,omitempty"`
	Location          map[string]float64 `json:"location,omitempty"`
	FinalLocation     map[string]float64 `json:"final_location,omitempty"`
}

func NewRideStatusConsumer(service *app.RideService, ch *amqp.Channel, wsManager *api.WSManager) *RideStatusConsumer {
	return &RideStatusConsumer{
		service:   service,
		channel:   ch,
		queue:     "ride_status",
		wsManager: wsManager,
	}
}

func (c *RideStatusConsumer) Start(ctx context.Context) error {
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
			c.handleStatusUpdate(ctx, msg)
		}
	}()
	log.Println("ride_status_updates consumer started")
	return nil
}

func (c *RideStatusConsumer) handleStatusUpdate(ctx context.Context, msg amqp.Delivery) {
	var update RideStatusUpdate
	if err := json.Unmarshal(msg.Body, &update); err != nil {
		log.Printf("[ride_status_updates] invalid JSON: %v", err)
		// Don't requeue malformed messages
		msg.Nack(false, false)
		return
	}

	// Get ride details
	ride, err := c.service.GetRideByID(ctx, update.RideID)
	if err != nil {
		log.Printf("[ride_status_updates] failed to get ride %s: %v", update.RideID, err)
		msg.Nack(false, true) // Requeue for retry
		return
	}

	// // Update ride status in database
	// if err := c.service.UpdateRideStatus(ctx, update.RideID, update.Status, update.DriverID); err != nil {
	// 	log.Printf("[ride_status_updates] failed to update ride status: %v", err)
	// 	msg.Nack(false, true)
	// 	return
	// }

	// // Update additional fields based on status
	// if update.Status == "IN_PROGRESS" && update.StartedAt != "" {
	// 	if err := c.service.UpdateRideStartTime(ctx, update.RideID, update.StartedAt); err != nil {
	// 		log.Printf("[ride_status_updates] failed to update start time: %v", err)
	// 	}
	// }

	// if update.Status == "COMPLETED" {
	// 	if err := c.service.UpdateRideCompletion(ctx, update.RideID, update.CompletedAt, update.FinalFare, update.ActualDistanceKm, update.ActualDurationMin); err != nil {
	// 		log.Printf("[ride_status_updates] failed to update completion: %v", err)
	// 	}
	// }

	// Record event in ride_events table
	eventData := map[string]interface{}{
		"status":    update.Status,
		"driver_id": update.DriverID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if update.FinalFare > 0 {
		eventData["final_fare"] = update.FinalFare
		eventData["actual_distance_km"] = update.ActualDistanceKm
		eventData["actual_duration_min"] = update.ActualDurationMin
	}

	// eventType := "RIDE_" + update.Status
	eventType := "RIDE_REQUESTED"
	if err := c.service.RecordEvent(ctx, update.RideID, eventType, eventData); err != nil {
		log.Printf("[ride_status_updates] failed to record event: %v", err)
	}

	// Send WebSocket update to passenger
	if c.wsManager != nil {
		wsUpdate := map[string]interface{}{
			"type":    "ride_status_update",
			"ride_id": update.RideID,
			"status":  update.Status,
		}

		switch update.Status {
		case "IN_PROGRESS":
			wsUpdate["message"] = "Your ride has started"
		case "COMPLETED":
			wsUpdate["message"] = "Your ride has been completed"
			wsUpdate["final_fare"] = update.FinalFare
		}

		if err := c.wsManager.SendToPassenger(ride.PassengerID, wsUpdate); err != nil {
			log.Printf("[ride_status_updates] failed to send to passenger: %v", err)
		}
	}

	msg.Ack(false)
}
