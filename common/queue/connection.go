package queue

import (
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"net/url"
	"sync"
	"time"
)

const reconnectBackoffBase = time.Millisecond * 100

type Connection struct {
	url url.URL

	mu      sync.RWMutex
	conn    *amqp.Connection
	closing chan *amqp.Error
}

func newConnection(url url.URL) (*Connection, error) {
	rc := &Connection{
		url: url,
	}

	if err := rc.setup(); err != nil {
		return nil, err
	}

	return rc, nil
}

func (rc *Connection) setup() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.conn != nil && !rc.conn.IsClosed() {
		err := rc.conn.Close()
		if err != nil {
			return errors.Wrap(err, "failed to close previous connection")
		}
	}

	conn, err := amqp.Dial(rc.url.String())
	if err != nil {
		return errors.Wrap(err, "failed to dial rabbitmq")
	}

	closing := make(chan *amqp.Error)
	conn.NotifyClose(closing)

	rc.conn = conn
	rc.closing = closing

	return nil
}

func (rc *Connection) closeListener() {
	backoff := reconnectBackoffBase
	for {
		<-rc.closing
		if err := rc.setup(); err != nil {
			log.Printf("Failed to reconnect to rabbitmq: %v", err)
			time.Sleep(backoff)
			backoff *= 2
		} else {
			backoff = reconnectBackoffBase
		}
	}
}

func (rc *Connection) isReady() bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return rc.conn != nil && !rc.conn.IsClosed()
}
