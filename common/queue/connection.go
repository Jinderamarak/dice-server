package queue

import (
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const reconnectBackoffBase = time.Millisecond * 100

type Connection struct {
	url          string
	channelCount uint32

	mu       sync.RWMutex
	inner    *amqp.Connection
	channels []*Channel
	cursor   atomic.Uint32
	closing  chan *amqp.Error
}

func newConnection(url string, channelCount uint32) (*Connection, error) {
	rc := &Connection{
		url:          url,
		channelCount: channelCount,
		channels:     make([]*Channel, channelCount),
	}

	if err := rc.setup(); err != nil {
		return nil, err
	}

	return rc, nil
}

func (rc *Connection) setup() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.inner != nil && !rc.inner.IsClosed() {
		err := rc.inner.Close()
		if err != nil {
			return errors.Wrap(err, "failed to close previous connection")
		}
	}

	conn, err := amqp.Dial(rc.url)
	if err != nil {
		return errors.Wrap(err, "failed to dial rabbitmq")
	}

	closing := make(chan *amqp.Error)
	conn.NotifyClose(closing)

	rc.inner = conn
	rc.closing = closing
	rc.cursor.Store(0)

	for i := uint32(0); i < rc.channelCount; i++ {
		ch, err := newChannel(rc)
		if err != nil {
			_ = conn.Close()
			return errors.Wrap(err, "failed to create channel")
		}
		rc.channels[i] = ch
	}

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

	return rc.inner != nil && !rc.inner.IsClosed()
}

func (rc *Connection) channel() *Channel {
	idx := rc.cursor.Add(1) % rc.channelCount
	return rc.channels[idx]
}

func (rc *Connection) exclusive() (*Channel, error) {
	ch, err := newChannel(rc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create exclusive channel")
	}
	return ch, nil
}

func (rc *Connection) Close() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	for _, ch := range rc.channels {
		if ch != nil {
			if err := ch.Close(); err != nil {
				return errors.Wrap(err, "failed to close channel")
			}
		}
	}

	if rc.inner != nil {
		if err := rc.inner.Close(); err != nil {
			return errors.Wrap(err, "failed to close connection")
		}
		rc.inner = nil
	}

	return nil
}
