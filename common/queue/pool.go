package queue

import (
	"github.com/pkg/errors"
	"sync/atomic"
)

type Pool struct {
	conns  []*Connection
	cursor atomic.Uint32
}

func NewPool(connections, channels uint32, url string) (*Pool, error) {
	pool := &Pool{}
	pool.conns = make([]*Connection, connections)
	for i := uint32(0); i < connections; i++ {
		rc, err := newConnection(url, channels)
		if err != nil {
			for j := uint32(0); j < i; j++ {
				_ = pool.conns[j].inner.Close()
			}
			return nil, errors.Wrapf(err, "failed to create rabbit connection at %d", i)
		}
		pool.conns[i] = rc
	}

	for _, rc := range pool.conns {
		go rc.closeListener()
	}
	return pool, nil
}

func (pool *Pool) connection() *Connection {
	for {
		idx := pool.cursor.Add(1) % uint32(len(pool.conns))
		conn := pool.conns[idx]
		if conn.isReady() {
			return conn
		}
	}
}

func (pool *Pool) channel() *Channel {
	return pool.connection().channel()
}

func (pool *Pool) exclusive() (*Channel, error) {
	return pool.connection().exclusive()
}

func (pool *Pool) GetConsumer(declaration Declaration) *Consumer {
	return newConsumer(pool, declaration)
}

func (pool *Pool) GetPublisher(declaration Declaration) *Publisher {
	return newPublisher(pool, declaration)
}

func (pool *Pool) Close() error {
	for _, rc := range pool.conns {
		if err := rc.Close(); err != nil {
			return errors.Wrap(err, "failed to close connection")
		}
	}
	return nil
}
