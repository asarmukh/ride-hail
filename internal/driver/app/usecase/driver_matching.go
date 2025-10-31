package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"ride-hail/internal/driver/adapter/psql"
	"ride-hail/internal/shared/util"

	"github.com/rabbitmq/amqp091-go"
)

type MatchingConsumer struct {
	service   Service
	channel   *amqp091.Channel
	repo      psql.Repo
	offers    map[string]*OfferState
	offersMux sync.RWMutex
	wsManager WSManager
}

// WSManager interface for WebSocket operations
type WSManager interface {
	SendOfferToDriver(driverID string, offer interface{}) error
	SendMessageToDriver(driverID string, message interface{}) error
}

type RideMatchRequest struct {
	RideID         string                 `json:"ride_id"`
	RideNumber     string                 `json:"ride_number"`
	PickupLocation map[string]interface{} `json:"pickup_location"`
	DestLocation   map[string]interface{} `json:"destination_location"`
	RideType       string                 `json:"ride_type"`
	EstimatedFare  float64                `json:"estimated_fare"`
	TimeoutSeconds int                    `json:"timeout_seconds"`
	CorrelationID  string                 `json:"correlation_id"`
}

type OfferState struct {
	RideID     string
	DriverID   string
	ExpiresAt  time.Time
	ResponseCh chan OfferResponse
}

type OfferResponse struct {
	Accepted bool
	Location LocationResponse
}

type LocationResponse struct {
	Latitude  float64
	Longitude float64
}

func NewMatchingConsumer(service Service, repo psql.Repo, channel *amqp091.Channel, wsManager WSManager) *MatchingConsumer {
	return &MatchingConsumer{
		service:   service,
		channel:   channel,
		repo:      repo,
		offers:    make(map[string]*OfferState),
		offersMux: sync.RWMutex{},
		wsManager: wsManager,
	}
}

func (c *MatchingConsumer) Start() error {
	// Declare queue
	_, err := c.channel.QueueDeclare(
		"driver_matching", // name
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Start consuming with manual acknowledgment
	msgs, err := c.channel.Consume(
		"driver_matching", // queue
		"",                // consumer
		false,             // auto-ack - DISABLED for manual ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Println("Driver matching consumer started, waiting for messages...")

	// Process messages
	go func() {
		for msg := range msgs {
			go c.handleRideRequest(msg)
		}
	}()

	return nil
}

func (c *MatchingConsumer) handleRideRequest(msg amqp091.Delivery) {
	ctx := context.Background()

	var request RideMatchRequest
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		log.Printf("Error unmarshaling ride request: %v", err)
		msg.Nack(false, false) // Don't requeue malformed messages
		return
	}

	log.Printf("Processing ride matching request: ride_id=%s, type=%s", request.RideID, request.RideType)

	// Extract pickup coordinates
	pickupLat, ok1 := request.PickupLocation["lat"].(float64)
	pickupLng, ok2 := request.PickupLocation["lng"].(float64)
	if !ok1 || !ok2 {
		log.Printf("Invalid pickup coordinates in request")
		msg.Nack(false, false)
		return
	}

	// Find nearby drivers
	drivers, err := c.repo.FindNearbyDrivers(ctx, pickupLat, pickupLng, request.RideType, 5.0)
	if err != nil {
		log.Printf("Error finding nearby drivers: %v", err)
		msg.Nack(false, true) // Requeue for retry
		return
	}

	if len(drivers) == 0 {
		log.Printf("No drivers available for ride %s", request.RideID)
		// TODO: Send "no drivers available" response to ride service
		msg.Ack(false)
		return
	}

	// Score and rank drivers
	rankedDrivers := c.rankDrivers(drivers)

	// Try to match with drivers sequentially (first-come-first-served)
	matched := false
	for _, driver := range rankedDrivers {
		log.Printf("Sending offer to driver %s (distance: %.2f km)", driver.ID, driver.DistanceKm)

		// Calculate ETA
		eta := c.calculateETA(driver.DistanceKm, 40.0) // Assume 40 km/h average speed

		// Generate unique offer ID
		offerID, _ := util.GenerateUUID()
		log.Printf("offerID: %s", offerID)
		// Create offer
		offer := map[string]interface{}{
			"type":        "ride_offer",
			"offer_id":    offerID,
			"ride_id":     request.RideID,
			"ride_number": request.RideNumber,
			"pickup_location": map[string]interface{}{
				"latitude":  request.PickupLocation["lat"],
				"longitude": request.PickupLocation["lng"],
				"address":   request.PickupLocation["address"],
			},
			"destination_location": map[string]interface{}{
				"latitude":  request.DestLocation["lat"],
				"longitude": request.DestLocation["lng"],
				"address":   request.DestLocation["address"],
			},
			"estimated_fare":        request.EstimatedFare,
			"estimated_distance_km": driver.DistanceKm,
			"expires_at":            time.Now().Add(30 * time.Second).Format(time.RFC3339),
		}

		// Send offer via WebSocket if manager is available
		if c.wsManager != nil {
			if err := c.wsManager.SendOfferToDriver(driver.ID, offer); err != nil {
				log.Printf("Error sending offer to driver %s: %v", driver.ID, err)
				continue
			}

			// Create offer state for tracking response
			responseCh := make(chan OfferResponse, 1)
			c.offersMux.Lock()
			c.offers[offerID] = &OfferState{
				RideID:     request.RideID,
				DriverID:   driver.ID,
				ExpiresAt:  time.Now().Add(60 * time.Second),
				ResponseCh: responseCh,
			}
			c.offersMux.Unlock()

			// Wait for response or timeout
			select {
			case response := <-responseCh:
				if response.Accepted {
					log.Printf("Driver %s accepted ride %s", driver.ID, request.RideID)

					// Send match response to ride service
					if err := c.sendMatchResponse(request, driver, eta); err != nil {
						log.Printf("Error sending match response: %v", err)
						continue
					}

					// Update driver status to EN_ROUTE
					if err := c.repo.UpdateDriverStatus(ctx, driver.ID, "EN_ROUTE"); err != nil {
						log.Printf("Error updating driver status: %v", err)
					}

					matched = true
					break
				} else {
					log.Printf("Driver %s rejected ride %s", driver.ID, request.RideID)
					// Try next driver
					continue
				}
			case <-time.After(60 * time.Second):
				log.Printf("Offer to driver %s timed out", driver.ID)
				// Clean up expired offer
				c.offersMux.Lock()
				delete(c.offers, offerID)
				c.offersMux.Unlock()
				// Try next driver
				continue
			}
		} else {
			// Fallback: auto-match if WebSocket is not available
			log.Printf("WebSocket manager not available, auto-matching with driver %s", driver.ID)

			// Send match response to ride service
			if err := c.sendMatchResponse(request, driver, eta); err != nil {
				log.Printf("Error sending match response: %v", err)
				continue
			}

			// Update driver status
			if err := c.repo.UpdateDriverStatus(ctx, driver.ID, "EN_ROUTE"); err != nil {
				log.Printf("Error updating driver status: %v", err)
			}

			matched = true
			break
		}

		if matched {
			break
		}
	}

	if matched {
		msg.Ack(false) // Acknowledge successful processing
	} else {
		msg.Nack(false, true) // Requeue if no driver matched
	}
}

func (c *MatchingConsumer) rankDrivers(drivers []psql.NearbyDriver) []psql.NearbyDriver {
	// Score drivers based on distance, rating, and completion rate
	type scoredDriver struct {
		driver psql.NearbyDriver
		score  float64
	}

	scored := make([]scoredDriver, len(drivers))
	for i, driver := range drivers {
		score := c.calculateDriverScore(driver)
		scored[i] = scoredDriver{driver: driver, score: score}
	}

	// Simple bubble sort by score (descending)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Extract drivers
	ranked := make([]psql.NearbyDriver, len(scored))
	for i, s := range scored {
		ranked[i] = s.driver
	}

	return ranked
}

func (c *MatchingConsumer) calculateDriverScore(driver psql.NearbyDriver) float64 {
	// Weight factors
	const (
		distanceWeight   = 0.5
		ratingWeight     = 0.3
		completionWeight = 0.2
	)

	// Normalize distance (inverse: closer = better)
	distanceScore := 1.0 / (1.0 + driver.DistanceKm)

	// Normalize rating (0-5 scale)
	ratingScore := driver.Rating / 5.0

	// Calculate completion rate
	completionRate := 0.0
	if driver.TotalRides > 0 {
		completionRate = float64(driver.CompletedRides) / float64(driver.TotalRides)
	}

	return (distanceScore * distanceWeight) +
		(ratingScore * ratingWeight) +
		(completionRate * completionWeight)
}

func (c *MatchingConsumer) calculateETA(distanceKm, avgSpeedKmh float64) time.Time {
	if avgSpeedKmh < 10 {
		avgSpeedKmh = 40.0 // Default to 40 km/h if speed is too low
	}

	durationHours := distanceKm / avgSpeedKmh
	durationMinutes := durationHours * 60

	return time.Now().Add(time.Duration(durationMinutes) * time.Minute)
}

func (c *MatchingConsumer) sendMatchResponse(request RideMatchRequest, driver psql.NearbyDriver, eta time.Time) error {
	response := map[string]interface{}{
		"ride_id":        request.RideID,
		"driver_id":      driver.ID,
		"accepted":       true,
		"correlation_id": request.CorrelationID,
		"driver_info": map[string]interface{}{
			"name":    driver.Email, // Using email as name placeholder
			"rating":  driver.Rating,
			"vehicle": driver.VehicleAttrs,
		},
		"driver_location": map[string]float64{
			"lat": driver.Latitude,
			"lng": driver.Longitude,
		},
		"estimated_arrival": eta.Format(time.RFC3339),
		"distance_km":       driver.DistanceKm,
	}

	body, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Publish to driver_topic exchange with ride-specific routing key
	routingKey := fmt.Sprintf("driver.response.%s", request.RideID)
	err = c.channel.Publish(
		"driver_topic", // exchange
		routingKey,     // routing key
		false,          // mandatory
		false,          // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish match response: %w", err)
	}

	log.Printf("Match response sent for ride %s to driver %s", request.RideID, driver.ID)
	return nil
}

// HandleDriverResponse processes responses from drivers (accept/reject)
func (c *MatchingConsumer) HandleDriverResponse(offerID string, accepted bool, location LocationResponse) error {
	c.offersMux.Lock()
	offer, exists := c.offers[offerID]
	if exists {
		delete(c.offers, offerID) // Clean up offer immediately
	}
	c.offersMux.Unlock()

	if !exists {
		log.Printf("Offer %s not found or already expired", offerID)
		return errors.New("offer not found or expired")
	}

	// Send response to waiting channel (non-blocking)
	select {
	case offer.ResponseCh <- OfferResponse{Accepted: accepted, Location: location}:
		log.Printf("Driver response sent to matching consumer for offer %s", offerID)
		return nil
	default:
		log.Printf("Offer response channel full for offer %s", offerID)
		return errors.New("offer response channel full")
	}
}
