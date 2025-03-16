package channel

import (
	"dice-server/internal/channel/message"
	"github.com/gorilla/websocket"
	"log"
	"sync/atomic"
	"time"
)

type WebSocketChannel struct {
	conn *websocket.Conn

	incoming chan *message.Message
	outgoing chan *message.Message
	closing  chan struct{}
	closed   atomic.Bool
}

func NewWebSocketChannel(conn *websocket.Conn) *WebSocketChannel {
	if conn == nil {
		panic("websocket connection is nil")
	}

	client := &WebSocketChannel{
		conn:     conn,
		incoming: make(chan *message.Message, MessageLimit),
		outgoing: make(chan *message.Message, MessageLimit),
		closing:  make(chan struct{}),
		closed:   atomic.Bool{},
	}

	go client.readingLoop()
	go client.writingLoop()
	return client
}

func (client *WebSocketChannel) readingLoop() {
	for {
		var msg message.Message
		err := client.conn.ReadJSON(&msg)

		switch {
		case websocket.IsUnexpectedCloseError(err):
			log.Println("unexpected close error:", err)
			client.Close()
			return
		case err != nil:
			log.Println("failed to read message:", err)
			continue
		}

		select {
		case client.incoming <- &msg:
		default:
			log.Println("dropped incoming message")
		}
	}
}

func (client *WebSocketChannel) writingLoop() {
	for {
		select {
		case <-client.closing:
			return
		case msg := <-client.outgoing:
			err := client.conn.WriteJSON(msg)
			switch {
			case websocket.IsUnexpectedCloseError(err):
				log.Println("unexpected close error:", err)
				client.Close()
				return
			case err != nil:
				log.Println("failed to write message:", err)
				continue
			}
		}
	}
}

func (client *WebSocketChannel) Close() {
	_ = client.conn.Close()
	if !client.closed.Swap(true) {
		close(client.closing)
	}
}

func (client *WebSocketChannel) WriteMessage(message *message.Message) error {
	select {
	case client.outgoing <- message:
		return nil
	default:
		return ErrMessageLimit
	}
}

func (client *WebSocketChannel) ReadMessage(timeout time.Duration) (*message.Message, error) {
	if timeout == 0 {
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
		return nil, ErrReadTimeout
	}
}

func (client *WebSocketChannel) ReadChannel() <-chan *message.Message {
	return client.incoming
}

func (client *WebSocketChannel) Closed() <-chan struct{} {
	return client.closing
}
