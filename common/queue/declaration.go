package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"time"
)

const temporaryTimeout = time.Second * 10

type Declaration struct {
	Name      string
	Temporary bool
	AutoAck   bool
	QoS       bool
}

func (decl *Declaration) declareQueue(channel *amqp.Channel) (amqp.Queue, error) {
	if decl.Temporary {
		return channel.QueueDeclare(
			decl.Name,
			false,
			false,
			false,
			false,
			amqp.Table{
				"x-expires": int32(temporaryTimeout.Milliseconds()),
			},
		)
	} else {
		return channel.QueueDeclare(
			decl.Name,
			true,
			false,
			false,
			false,
			nil,
		)
	}
}
