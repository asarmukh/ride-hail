package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-hail/internal/ride/domain"
	"ride-hail/internal/shared/models"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch *amqp091.Channel
}

func NewPublisher(ch *amqp091.Channel) *Publisher {
	return &Publisher{ch: ch}
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
				return conn, ch, nil
			}
		}
		log.Printf("RabbitMQ not ready, retrying... (%d/10)", i+1)
		time.Sleep(3 * time.Second)
	}

	return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
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
