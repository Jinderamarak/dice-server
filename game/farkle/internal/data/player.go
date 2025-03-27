package data

import (
	"dice-server/game/common/client"
	"log"
	"sync/atomic"
	"time"
)

const messageLimit = 16

type playerChannel interface {
	Send(message *client.Message) error
	Receive(timeout time.Duration) (*client.Message, error)
	Consume() <-chan *client.Message
	Closing() <-chan struct{}
}

type PlayerClient struct {
	channel playerChannel

	hasTurn  atomic.Bool
	incoming chan *client.Message

	gameStateHandler *func() *GameState
}

func NewPlayerClient(ch playerChannel) *PlayerClient {
	p := &PlayerClient{
		channel:  ch,
		hasTurn:  atomic.Bool{},
		incoming: make(chan *client.Message, messageLimit),
	}

	go p.readingLoop()
	return p
}

func (c *PlayerClient) readingLoop() {
	for {
		select {
		case <-c.channel.Closing():
			return
		case msg, ok := <-c.channel.Consume():
			if !ok {
				//	underlying channel closed
				return
			}

			if msg.Variant == VarPleaseSync {
				if c.gameStateHandler == nil {
					continue
				}

				state := (*c.gameStateHandler)()
				if state == nil {
					continue
				}

				err := c.Send(CraftSyncState(state))
				if err != nil {
					log.Println("Error sending sync state:", err)
				}
				continue
			}

			if !c.hasTurn.Load() {
				continue
			}

			select {
			case c.incoming <- msg:
			default:
				log.Println("Player channel is full, dropping message")
			}
		}
	}
}

func (c *PlayerClient) Send(message *client.Message) error {
	return c.channel.Send(message)
}

func (c *PlayerClient) Receive(timeout time.Duration) (*client.Message, error) {
	if timeout < 0 {
		select {
		case <-c.channel.Closing():
			return nil, client.ErrClientClosed
		case msg := <-c.incoming:
			return msg, nil
		}
	}

	select {
	case <-c.channel.Closing():
		return nil, client.ErrClientClosed
	case msg := <-c.incoming:
		return msg, nil
	case <-time.After(timeout):
		return nil, client.ErrRecvTimeout
	}
}

func (c *PlayerClient) Receiving() <-chan *client.Message {
	return c.incoming
}

func (c *PlayerClient) Closing() <-chan struct{} {
	return c.channel.Closing()
}

func (c *PlayerClient) SetTurn(isMyTurn bool) {
	c.hasTurn.Store(isMyTurn)
}

func (c *PlayerClient) SetGameStateHandler(handler func() *GameState) {
	c.gameStateHandler = &handler
}
