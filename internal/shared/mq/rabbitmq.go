package mq

import (
	"fmt"
	"github.com/rabbitmq/amqp091-go"
	"ride-hail/internal/shared/models"
)

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
