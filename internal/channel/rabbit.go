package channel

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"sync/atomic"
	"time"
)

type RabbitChannel struct {
	channel    *amqp.Channel
	writeQueue amqp.Queue
	readQueue  amqp.Queue
	consumer   <-chan amqp.Delivery

	incoming chan *Message
	outgoing chan *Message
	closing  chan struct{}
	closed   atomic.Bool
}

func OpenRabbitChannel(conn *amqp.Connection, writeTopic, readTopic string) (*RabbitChannel, error) {
	if conn == nil {
		panic("rabbitmq connection is nil")
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	writeQueue, err := declareWriteQueue(channel, writeTopic)
	if err != nil {
		return nil, err
	}

	readQueue, err := declareReadQueue(channel, readTopic)
	if err != nil {
		return nil, err
	}

	consumer, err := channel.Consume(
		readQueue.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	client := &RabbitChannel{
		channel:    channel,
		writeQueue: writeQueue,
		readQueue:  readQueue,
		consumer:   consumer,
		incoming:   make(chan *Message, MessageLimit),
		outgoing:   make(chan *Message, MessageLimit),
		closing:    make(chan struct{}),
		closed:     atomic.Bool{},
	}

	go client.readingLoop()
	go client.writingLoop()
	return client, nil
}

func declareWriteQueue(channel *amqp.Channel, topic string) (amqp.Queue, error) {
	return channel.QueueDeclare(
		topic,
		false,
		false,
		false,
		false,
		nil,
	)
}

func declareReadQueue(channel *amqp.Channel, topic string) (amqp.Queue, error) {
	return channel.QueueDeclare(
		topic,
		false,
		false,
		false,
		false,
		nil,
	)
}

func (client *RabbitChannel) readingLoop() {
	for {
		select {
		case <-client.closing:
			return
		case delivery, ok := <-client.consumer:
			if !ok {
				log.Println("consumer channel is closed")
				client.Close()
				return
			}

			var message Message
			err := message.Unmarshal(delivery.Body)
			if err != nil {
				log.Println("failed to unmarshal message:", err)
				continue
			}

			select {
			case client.incoming <- &message:
			default:
				log.Println("dropped incoming message")
			}
		}
	}
}

func (client *RabbitChannel) writingLoop() {
	for {
		select {
		case <-client.closing:
			return
		case message := <-client.outgoing:
			data, err := message.Marshal()
			if err != nil {
				log.Println("failed to marshal message:", err)
				continue
			}

			err = client.channel.Publish(
				"",
				client.writeQueue.Name,
				false,
				false,
				amqp.Publishing{
					ContentType: "application/json",
					Body:        data,
				})
			if err != nil {
				log.Println("failed to publish message:", err)
				continue
			}
		}
	}
}

func (client *RabbitChannel) Close() {
	_ = client.channel.Close()
	if !client.closed.Swap(true) {
		close(client.closing)
	}
}

func (client *RabbitChannel) SendMessage(message *Message) error {
	select {
	case client.outgoing <- message:
		return nil
	default:
		return ErrMessageLimit
	}
}

func (client *RabbitChannel) ReadMessage(timeout time.Duration) (*Message, error) {
	if timeout == 0 {
		select {
		case <-client.closing:
			return nil, ErrClientClosed
		case message := <-client.incoming:
			return message, nil
		}
	}

	select {
	case <-client.closing:
		return nil, ErrClientClosed
	case message := <-client.incoming:
		return message, nil
	case <-time.After(timeout):
		return nil, ErrReadTimeout
	}
}

func (client *RabbitChannel) ReadChannel() <-chan *Message {
	return client.incoming
}

func (client *RabbitChannel) Closed() <-chan struct{} {
	return client.closing
}
