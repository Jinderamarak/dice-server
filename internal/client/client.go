package client

import (
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"time"
)

var reconnectionTimeout = time.Second * 30

var ErrReconnectionTimeout = errors.New("reconnection timeout")

type Client struct {
	conn        *websocket.Conn
	session     uuid.UUID
	reconnected chan *websocket.Conn
	termination chan<- uuid.UUID
}

func newClient(conn *websocket.Conn, session uuid.UUID, termination chan<- uuid.UUID) *Client {
	return &Client{
		conn:        conn,
		session:     session,
		reconnected: make(chan *websocket.Conn),
		termination: termination,
	}
}

func (client *Client) NotifyReconnection(conn *websocket.Conn) {
	client.reconnected <- conn
}

func (client *Client) Terminate() error {
	client.termination <- client.session
	return client.conn.Close()
}

func (client *Client) waitForReconnection() error {
	select {
	case conn := <-client.reconnected:
		client.conn = conn
		return nil
	case <-time.After(reconnectionTimeout):
		return ErrReconnectionTimeout
	}
}

func (client *Client) SendMessage(message Message) error {
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

func (client *Client) ReadMessage() (Message, error) {
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
