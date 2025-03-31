package queue

import (
	"github.com/pkg/errors"
	"net/url"
	"sync/atomic"
)

type Pool struct {
	conns  []*Connection
	cursor atomic.Uint32
}

func NewPool(size int, url url.URL) (*Pool, error) {
	pool := &Pool{}
	pool.conns = make([]*Connection, size)
	for i := 0; i < size; i++ {
		rc, err := newConnection(url)
		if err != nil {
			for j := 0; j < i; j++ {
				_ = pool.conns[j].conn.Close()
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

func (pool *Pool) get() *Connection {
	for {
		idx := pool.cursor.Add(1) % uint32(len(pool.conns))
		conn := pool.conns[idx]
		if conn.isReady() {
			return conn
		}
	}
}

func (pool *Pool) GetConsumer(declaration Declaration) *Consumer {
	return newConsumer(pool, declaration)
}

func (pool *Pool) GetPublisher(declaration Declaration) *Publisher {
	return newPublisher(pool, declaration)
}

func (pool *Pool) Close() error {
	for _, rc := range pool.conns {
		rc.mu.Lock()
		_ = rc.conn.Close()
		rc.mu.Unlock()
	}
	return nil
}
