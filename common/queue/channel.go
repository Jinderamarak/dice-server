package queue

import (
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"sync"
	"time"
)

type Channel struct {
	conn *Connection

	mu      sync.RWMutex
	inner   *amqp.Channel
	closing chan *amqp.Error
}

func newChannel(conn *Connection) (*Channel, error) {
	c := &Channel{
		conn: conn,
	}

	if err := c.setup(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Channel) setup() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.inner != nil && !c.inner.IsClosed() {
		err := c.inner.Close()
		if err != nil {
			return errors.Wrap(err, "failed to close previous channel")
		}
	}

	ch, err := c.conn.inner.Channel()
	if err != nil {
		return errors.Wrap(err, "failed to create channel")
	}

	closing := make(chan *amqp.Error)
	ch.NotifyClose(closing)

	c.inner = ch
	c.closing = closing

	return nil
}

func (c *Channel) closeListener() {
	backoff := reconnectBackoffBase
	for {
		<-c.closing
		if !c.conn.isReady() {
			return
		}

		if err := c.setup(); err != nil {
			log.Printf("Failed to recreate channel: %v", err)
			time.Sleep(backoff)
			backoff *= 2
		} else {
			backoff = reconnectBackoffBase
		}
	}
}

func (c *Channel) isReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.inner != nil && !c.inner.IsClosed()
}

func (c *Channel) lock() {
	c.mu.Lock()
}

func (c *Channel) unlock() {
	c.mu.Unlock()
}

func (c *Channel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.inner != nil {
		if err := c.inner.Close(); err != nil {
			return errors.Wrap(err, "failed to close channel")
		}
		c.inner = nil
	}

	return nil
}
