package queue

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
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

func (cons *Consumer) listen(conn *Connection) error {
	var ch *Channel
	if cons.declaration.QoS {
		exc, err := cons.pool.exclusive()
		if err != nil {
			return errors.Wrap(err, "failed to create exclusive channel")
		}
		ch = exc
	} else {
		ch = conn.channel()
	}

	ch.lock()
	que, err := cons.declaration.declareQueue(ch.inner)
	ch.unlock()

	if err != nil {
		return errors.Wrap(err, "failed to declare a queue")
	}

	if cons.declaration.QoS {
		if err = ch.inner.Qos(
			1,
			0,
			false,
		); err != nil {
			return errors.Wrap(err, "failed to set QoS")
		}
	}

	consumerName := uuid.New().String()
	ch.lock()
	msgs, err := ch.inner.Consume(
		que.Name,
		consumerName,
		cons.declaration.AutoAck,
		false,
		false,
		false,
		nil,
	)
	ch.unlock()

	if err != nil {
		return errors.Wrap(err, "failed to consume messages")
	}

	defer func() {
		ch.lock()
		if err := ch.inner.Cancel(consumerName, true); err != nil {
			log.Println("Failed to cancel consumer:", err)
		}
		ch.unlock()
	}()

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
		conn := cons.pool.connection()
		err := cons.listen(conn)
		if err != nil {
			log.Println("Consumer failed at listening:", err)
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
