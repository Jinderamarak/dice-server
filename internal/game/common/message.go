package common

import "github.com/google/uuid"

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
