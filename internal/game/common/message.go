package common

import (
	"dice-server/internal/client"
	"github.com/google/uuid"
)

const (
	VarGameBegin = "game-begin"
	VarGameEnd   = "game-end"
	VarError     = "error"
)

type VariantGameBegin struct {
	Players []uuid.UUID `json:"players"`
}

type VariantGameEnd struct {
	Winners []uuid.UUID `json:"winners"`
}

type VariantError struct {
	Message string `json:"message"`
}

func MakeGameBegin(players ...uuid.UUID) *client.Message {
	return client.MustMarshalMessage(VarGameBegin, VariantGameBegin{Players: players})
}

func MakeGameEnd(winners ...uuid.UUID) *client.Message {
	return client.MustMarshalMessage(VarGameEnd, VariantGameEnd{Winners: winners})
}

func MakeError(message string) *client.Message {
	return client.MustMarshalMessage(VarError, VariantError{Message: message})
}
