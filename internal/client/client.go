package client

import (
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"sync/atomic"
	"time"
)

const messageLimit = 32

var reconnectionTimeout = time.Second * 60

var ErrReconnectionTimeout = errors.New("reconnection timeout")
var ErrClientClosed = errors.New("client is closed")
var ErrReadTimeout = errors.New("read timeout")

type WebSocketClient struct {
	conn    *websocket.Conn
	session uuid.UUID

	reconnect   chan *websocket.Conn
	reconnected chan struct{}
	reconnectEx atomic.Bool

	closure chan<- uuid.UUID
	closed  atomic.Bool

	incoming chan *Message
	outgoing chan *Message
	errors   chan error
}

func newClient(conn *websocket.Conn, session uuid.UUID, closure chan<- uuid.UUID) *WebSocketClient {
	client := &WebSocketClient{
		conn:        conn,
		session:     session,
		reconnect:   make(chan *websocket.Conn, 1),
		reconnected: make(chan struct{}),
		closure:     closure,
		closed:      atomic.Bool{},
		incoming:    make(chan *Message, messageLimit),
		outgoing:    make(chan *Message, messageLimit),
		errors:      make(chan error),
	}

	go client.readingLoop()
	go client.writingLoop()
	return client
}

func (client *WebSocketClient) readingLoop() {
	for {
		if client.closed.Load() {
			return
		}

		var message Message
		if err := client.conn.ReadJSON(&message); websocket.IsUnexpectedCloseError(err) {
			log.Println("Reading message failed:", err)

			if err := client.waitForReconnection(); err != nil {
				client.errors <- err
				client.Close()
				return
			}
			continue
		}

		select {
		case client.incoming <- &message:
		default:
			log.Println("dropped incoming message")
		}
	}
}

func (client *WebSocketClient) writingLoop() {
	for {
		if client.closed.Load() {
			return
		}

		message := <-client.outgoing
		if err := client.conn.WriteJSON(message); websocket.IsUnexpectedCloseError(err) {
			log.Println("Sending message failed:", err)

			select {
			case <-client.reconnected:
				if err := client.conn.WriteJSON(message); err != nil {
					client.errors <- err
					client.Close()
					return
				}
			case <-time.After(reconnectionTimeout):
				if err := client.conn.WriteJSON(message); err != nil {
					client.errors <- err
					client.Close()
					return
				}
			}
		}
	}
}

func (client *WebSocketClient) waitForReconnection() error {
	if client.closed.Load() {
		return ErrClientClosed
	}

	log.Println("Waiting for reconnection...")
	select {
	case conn := <-client.reconnect:
		client.conn = conn
		close(client.reconnected)
		client.reconnected = make(chan struct{})
		return nil
	case <-time.After(reconnectionTimeout):
		return ErrReconnectionTimeout
	}
}

func (client *WebSocketClient) NotifyReconnection(conn *websocket.Conn) error {
	if client.closed.Load() {
		return ErrClientClosed
	}

	select {
	case <-client.reconnected:
		//	drained previous
	default:
		//	nothing to drain
	}

	client.reconnect <- conn
	return nil
}

func (client *WebSocketClient) Close() {
	client.closed.Store(true)
	_ = client.conn.Close()

	client.closure <- client.session
}

func (client *WebSocketClient) SendMessage(message *Message) {
	client.outgoing <- message
}

func (client *WebSocketClient) ReadMessage(timeout time.Duration) (*Message, error) {
	if timeout == 0 {
		select {
		case message := <-client.incoming:
			return message, nil
		case err := <-client.errors:
			return nil, err
		}
	}

	select {
	case message := <-client.incoming:
		return message, nil
	case err := <-client.errors:
		return nil, err
	case <-time.After(timeout):
		return nil, ErrReadTimeout
	}
}
