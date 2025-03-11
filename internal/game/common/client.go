package common

import "dice-server/internal/client"

type Client interface {
	SendMessage(message client.Message) error
	ReadMessage() (client.Message, error)
}
