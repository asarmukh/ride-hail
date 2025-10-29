package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"ride-hail/internal/ride/domain"
	"ride-hail/internal/shared/models"

	"github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch  *amqp091.Channel
	mu  sync.RWMutex
	url string
}

func NewPublisher(ch *amqp091.Channel) *Publisher {
	return &Publisher{ch: ch}
}

// RabbitMQConnection manages RabbitMQ connection with auto-reconnection
type RabbitMQConnection struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	url     string
	mu      sync.RWMutex
	done    chan struct{}
}

func ConnectToRMQ(cfg *models.RabbitMQConfig) (*amqp091.Connection, *amqp091.Channel, error) {
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.User, cfg.Password, cfg.Host, cfg.Port)

	var conn *amqp091.Connection
	var ch *amqp091.Channel
	var err error

	for i := 0; i < 10; i++ {
		conn, err = amqp091.Dial(dsn)
		if err == nil {
			ch, err = conn.Channel()
			if err == nil {
				// Start reconnection monitor
				go monitorConnection(conn, ch, dsn)
				return conn, ch, nil
			}
		}
		log.Printf("RabbitMQ not ready, retrying... (%d/10)", i+1)
		time.Sleep(3 * time.Second)
	}

	return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
}

// monitorConnection monitors the connection and reconnects if necessary
func monitorConnection(conn *amqp091.Connection, ch *amqp091.Channel, url string) {
	notifyClose := make(chan *amqp091.Error)
	conn.NotifyClose(notifyClose)

	for {
		err := <-notifyClose
		if err == nil {
			// Connection closed cleanly
			return
		}

		log.Printf("RabbitMQ connection lost: %v. Attempting to reconnect...", err)

		// Exponential backoff
		backoff := 5 * time.Second
		maxBackoff := 60 * time.Second

		for {
			time.Sleep(backoff)

			newConn, newErr := amqp091.Dial(url)
			if newErr != nil {
				log.Printf("Reconnection failed: %v. Retrying in %v...", newErr, backoff)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}

			newCh, newErr := newConn.Channel()
			if newErr != nil {
				newConn.Close()
				log.Printf("Failed to create channel: %v. Retrying in %v...", newErr, backoff)
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			}

			log.Println("Successfully reconnected to RabbitMQ")

			// Update references (note: this won't update existing Publisher instances)
			conn = newConn
			ch = newCh

			// Setup new close notification
			notifyClose = make(chan *amqp091.Error)
			conn.NotifyClose(notifyClose)
			break
		}
	}
}

func (p *Publisher) Publish(ctx context.Context, exchange, routingKey string, body []byte) error {
	err := p.ch.PublishWithContext(ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent,
			Timestamp:    time.Now(),
		})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

func (p *Publisher) PublishRideStatus(event domain.RideStatusEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(
		context.Background(),
		"ride_topic",            // exchange
		"ride.status.cancelled", // routing key
		false, false,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent,
		},
	)
}

func (p *Publisher) PublishFanout(ctx context.Context, exchange string, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return p.ch.PublishWithContext(ctx,
		exchange, // exchange
		"",       // routing key (empty for fanout)
		false,    // mandatory
		false,    // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent,
			Timestamp:    time.Now(),
		})
}
