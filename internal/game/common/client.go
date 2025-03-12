package common

import (
	"dice-server/internal/client"
	"time"
)

type Client interface {
	SendMessage(message *client.Message)
	ReadMessage(timeout time.Duration) (*client.Message, error)
	Close()
}
