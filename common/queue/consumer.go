package queue

import (
	"dice-server/common/utility"
	"fmt"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	pool        *Pool
	declaration Declaration
	msgs        chan amqp.Delivery
	closed      chan struct{}
}

func newConsumer(pool *Pool, queue Declaration) *Consumer {
	cons := &Consumer{
		pool:        pool,
		declaration: queue,
		msgs:        make(chan amqp.Delivery),
		closed:      make(chan struct{}),
	}

	go cons.loop()
	return cons
}

func (cons *Consumer) listen(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to open a channel")
	}
	defer utility.CloseAndIgnore(ch)

	que, err := cons.declaration.declareQueue(ch)
	if err != nil {
		return errors.Wrap(err, "failed to declare a queue")
	}

	if cons.declaration.QoS {
		if err = ch.Qos(
			1,
			0,
			false,
		); err != nil {
			return errors.Wrap(err, "failed to set QoS")
		}
	}

	msgs, err := ch.Consume(
		que.Name,
		"",
		cons.declaration.AutoAck,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "failed to consume messages")
	}

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return errors.New("channel closed")
			}
			cons.msgs <- msg
		case <-cons.closed:
			return nil
		}
	}
}

func (cons *Consumer) loop() {
	for {
		c := cons.pool.get()
		err := cons.listen(c.conn)
		if err != nil {
			fmt.Println("Consumer failed at listening:", err)
		} else {
			return
		}

		select {
		case <-cons.closed:
			return
		default:
			continue
		}
	}
}

func (cons *Consumer) Consume() <-chan amqp.Delivery {
	return cons.msgs
}

func (cons *Consumer) Close() {
	select {
	case <-cons.closed:
	default:
		close(cons.closed)
	}
}
