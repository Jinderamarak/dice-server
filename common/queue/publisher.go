package queue

import (
	"dice-server/common/utility"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rawMessage struct {
	body        []byte
	contentType string
}

type Publisher struct {
	pool        *Pool
	declaration Declaration
	msgs        chan *rawMessage
	closed      chan struct{}
}

func newPublisher(pool *Pool, queue Declaration) *Publisher {
	pub := &Publisher{
		pool:        pool,
		declaration: queue,
		msgs:        make(chan *rawMessage),
		closed:      make(chan struct{}),
	}

	go pub.loop()
	return pub
}

func (pub *Publisher) keepPublishing(conn *amqp.Connection, lastMsg **rawMessage) error {
	ch, err := conn.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to open a channel")
	}
	defer utility.CloseAndIgnore(ch)

	que, err := pub.declaration.declareQueue(ch)
	if err != nil {
		return errors.Wrap(err, "failed to declare a queue")
	}

	deliveryMode := amqp.Transient
	if !pub.declaration.Temporary {
		deliveryMode = amqp.Persistent
	}
	for {
		msg := *lastMsg
		if msg == nil {
			select {
			case msg = <-pub.msgs:
			case <-pub.closed:
				return nil
			}
		}

		if err := ch.Publish(
			"",
			que.Name,
			false,
			false,
			amqp.Publishing{
				DeliveryMode: deliveryMode,
				ContentType:  msg.contentType,
				Body:         msg.body,
			},
		); err != nil {
			lastMsg = &msg
			return errors.Wrap(err, "failed to publish message")
		}
	}
}

func (pub *Publisher) loop() {
	var lastMsg *rawMessage
	for {
		c := pub.pool.get()
		err := pub.keepPublishing(c.conn, &lastMsg)
		if err != nil {
			fmt.Println("Failed to publish message:", err)
		} else {
			return
		}

		select {
		case <-pub.closed:
			return
		default:
			continue
		}
	}
}

func (pub *Publisher) Publish(body []byte, contentType string) {
	pub.msgs <- &rawMessage{
		body:        body,
		contentType: contentType,
	}
}

func (pub *Publisher) PublishJSON(data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal json")
	}

	pub.Publish(body, "application/json")
	return nil
}

func (pub *Publisher) Close() {
	select {
	case <-pub.closed:
	default:
		close(pub.closed)
	}
}
