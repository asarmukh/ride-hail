package mq

import (
	"context"
	"fmt"
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

func ConnentToRMQ(cfg *models.RabbitMQConfig) (*amqp091.Connection, *amqp091.Channel, error) {
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.User, cfg.Password, cfg.Host, cfg.Port)

	conn, err := amqp091.Dial(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return conn, ch, nil
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
