package client

import (
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"time"
)

var reconnectionTimeout = time.Second * 30

var ErrReconnectionTimeout = errors.New("reconnection timeout")

type WebSocketClient struct {
	conn        *websocket.Conn
	session     uuid.UUID
	reconnected chan *websocket.Conn
	termination chan<- uuid.UUID
}

func newClient(conn *websocket.Conn, session uuid.UUID, termination chan<- uuid.UUID) *WebSocketClient {
	return &WebSocketClient{
		conn:        conn,
		session:     session,
		reconnected: make(chan *websocket.Conn),
		termination: termination,
	}
}

func (client *WebSocketClient) NotifyReconnection(conn *websocket.Conn) {
	client.reconnected <- conn
}

func (client *WebSocketClient) Terminate() error {
	client.termination <- client.session
	return client.conn.Close()
}

func (client *WebSocketClient) waitForReconnection() error {
	select {
	case conn := <-client.reconnected:
		client.conn = conn
		return nil
	case <-time.After(reconnectionTimeout):
		return ErrReconnectionTimeout
	}
}

func (client *WebSocketClient) SendMessage(message Message) error {
	err := client.conn.WriteJSON(message)
	switch {
	case errors.Is(err, websocket.ErrCloseSent):
		if err = client.waitForReconnection(); err != nil {
			return err
		}
		if err = client.conn.WriteJSON(message); err != nil {
			return err
		}
	case err != nil:
		return err
	}
	return nil
}

func (client *WebSocketClient) ReadMessage() (Message, error) {
	var message Message
	err := client.conn.ReadJSON(&message)
	switch {
	case errors.Is(err, websocket.ErrCloseSent):
		if err = client.waitForReconnection(); err != nil {
			return Message{}, err
		}
		if err = client.conn.ReadJSON(&message); err != nil {
			return Message{}, err
		}
	case err != nil:
		return Message{}, err
	}
	return message, nil
}
