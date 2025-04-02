package client

import (
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

const (
	messageBufferLimit = 32
	messageLengthLimit = 4096
)

var (
	ErrMessageLimit = errors.New("reached message limit")
	ErrClientClosed = errors.New("client was closed")
	ErrRecvTimeout  = errors.New("receive timeout")
)

type WebSocketClient struct {
	conn        *websocket.Conn
	reconnected chan struct{}
	closing     chan struct{}

	incoming chan *Message
	outgoing chan *Message
}

func NewWebSocketClient() *WebSocketClient {
	return &WebSocketClient{
		conn:        nil,
		reconnected: make(chan struct{}),
		closing:     make(chan struct{}),
		incoming:    make(chan *Message, messageBufferLimit),
		outgoing:    make(chan *Message, messageBufferLimit),
	}
}

func (client *WebSocketClient) readingLoop(conn *websocket.Conn) {
	conn.SetReadLimit(messageLengthLimit)

	for {
		var msg Message
		err := conn.ReadJSON(&msg)

		if err != nil {
			log.Println("Failed to read message:", err)
			return
		}

		select {
		case client.incoming <- &msg:
			//	message received
		default:
			//	message dropped
		}
	}
}

func (client *WebSocketClient) writingLoop(conn *websocket.Conn) {
	for {
		select {
		case msg := <-client.outgoing:
			err := conn.WriteJSON(msg)

			if err != nil {
				log.Println("Failed to write message:", err)
				return
			}
		case <-client.reconnected:
			return
		case <-client.closing:
			return
		}
	}
}

func (client *WebSocketClient) reconnect(conn *websocket.Conn) {
	if client.conn != nil {
		_ = client.conn.Close()
	}

	select {
	case <-client.reconnected:
	default:
		close(client.reconnected)
	}
	client.reconnected = make(chan struct{})

	client.conn = conn
	go client.readingLoop(conn)
	go client.writingLoop(conn)
}

func (client *WebSocketClient) close() {
	select {
	case <-client.closing:
	default:
		close(client.closing)
	}
	if client.conn != nil {
		_ = client.conn.Close()
	}
}

func (client *WebSocketClient) Send(msg *Message) error {
	select {
	case client.outgoing <- msg:
		return nil
	default:
		return ErrMessageLimit
	}
}

func (client *WebSocketClient) Receive(timeout time.Duration) (*Message, error) {
	if timeout < 0 {
		select {
		case <-client.closing:
			return nil, ErrClientClosed
		case msg := <-client.incoming:
			return msg, nil
		}
	}

	select {
	case <-client.closing:
		return nil, ErrClientClosed
	case msg := <-client.incoming:
		return msg, nil
	case <-time.After(timeout):
		return nil, ErrRecvTimeout
	}
}

func (client *WebSocketClient) Consume() <-chan *Message {
	return client.incoming
}

func (client *WebSocketClient) Closing() <-chan struct{} {
	return client.closing
}
