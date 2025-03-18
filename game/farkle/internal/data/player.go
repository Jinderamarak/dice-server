package data

import (
	"dice-server/common/channel"
	"dice-server/common/channel/message"
	"github.com/google/uuid"
	"log"
	"sync/atomic"
	"time"
)

const (
	PlayerStateDisconnected = iota
	PlayerStateConnected
)

type playerChannel interface {
	Close()
	SendMessage(message *message.Message) error
	ReadMessage(timeout time.Duration) (*message.Message, error)
	ReadChannel() <-chan *message.Message
	Closed() <-chan struct{}
}

type PlayerClient struct {
	id      uuid.UUID
	channel playerChannel

	state    atomic.Int32
	hasTurn  atomic.Bool
	incoming chan *message.Message

	importantHandler *func(*message.Message)
}

func NewPlayerClient(playerId uuid.UUID, ch playerChannel) *PlayerClient {
	p := &PlayerClient{
		id:       playerId,
		channel:  ch,
		state:    atomic.Int32{},
		hasTurn:  atomic.Bool{},
		incoming: make(chan *message.Message, channel.MessageLimit),
	}

	go p.readingLoop()
	return p
}

func (c *PlayerClient) readingLoop() {
	for {
		select {
		case <-c.channel.Closed():
			c.state.Store(PlayerStateDisconnected)
			return
		case msg := <-c.channel.ReadChannel():
			if msg == nil {
				c.Close()
				return
			}

			c.updateStateWithMessage(msg)

			if IsImportantMessage(msg) {
				if c.importantHandler != nil {
					(*c.importantHandler)(msg)
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

func (c *PlayerClient) updateStateWithMessage(msg *message.Message) {
	switch msg.Variant {
	case message.VarControlConnected:
		var data message.VariantControlConnected
		if err := msg.UnmarshalData(&data); err != nil {
			log.Println("Error unmarshalling control message:", err)
			return
		}

		if data.UserId == c.id {
			c.state.Store(PlayerStateConnected)
		}
	case message.VarControlDisconnected:
		var data message.VariantControlDisconnected
		if err := msg.UnmarshalData(&data); err != nil {
			log.Println("Error unmarshalling control message:", err)
			return
		}

		if data.UserId == c.id {
			c.state.Store(PlayerStateDisconnected)
		}
	}
}

func (c *PlayerClient) Close() {
	c.state.Store(PlayerStateDisconnected)
	c.channel.Close()
}

func (c *PlayerClient) SendMessage(message *message.Message) error {
	return c.channel.SendMessage(message)
}

func (c *PlayerClient) ReadMessage(timeout time.Duration) (*message.Message, error) {
	if timeout < 0 {
		select {
		case <-c.channel.Closed():
			return nil, channel.ErrClientClosed
		case msg := <-c.incoming:
			return msg, nil
		}
	}

	select {
	case <-c.channel.Closed():
		return nil, channel.ErrClientClosed
	case msg := <-c.incoming:
		return msg, nil
	case <-time.After(timeout):
		return nil, channel.ErrReadTimeout
	}
}

func (c *PlayerClient) ReadChannel() <-chan *message.Message {
	return c.incoming
}

func (c *PlayerClient) Closed() <-chan struct{} {
	return c.channel.Closed()
}

func (c *PlayerClient) GetState() int32 {
	return c.state.Load()
}

func (c *PlayerClient) SetOnTurn() {
	c.hasTurn.Store(true)
}

func (c *PlayerClient) SetOffTurn() {
	c.hasTurn.Store(false)
}

func (c *PlayerClient) SetImportantHandler(handler *func(*message.Message)) {
	c.importantHandler = handler
}
