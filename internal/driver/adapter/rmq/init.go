package rmq

import (
	"github.com/rabbitmq/amqp091-go"
)

type broker struct {
	ch *amqp091.Channel
}

type Broker interface {
}

func NewBroker(ch *amqp091.Channel) Broker {
	return &broker{ch: ch}
}
