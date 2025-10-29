package rmq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type broker struct {
	ch *amqp091.Channel
}

type Broker interface {
	PublishFanout(ctx context.Context, exchange string, data interface{}) error
	Publish(ctx context.Context, exchange, routingKey string, data interface{}) error
}

func NewBroker(ch *amqp091.Channel) Broker {
	return &broker{ch: ch}
}

func (b *broker) PublishFanout(ctx context.Context, exchange string, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return b.ch.PublishWithContext(ctx,
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

func (b *broker) Publish(ctx context.Context, exchange, routingKey string, data interface{}) error {
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return b.ch.PublishWithContext(ctx,
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
}
