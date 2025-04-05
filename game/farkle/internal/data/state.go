package data

import (
	"dice-server/game/farkle/connect"
	"github.com/google/uuid"
	"time"
)

type GameState struct {
	ID            uuid.UUID      `json:"id"`
	Target        int            `json:"target"`
	PickTime      time.Duration  `json:"pickTime"`
	CurrentPlayer uuid.UUID      `json:"currentPlayer"`
	Players       []*PlayerState `json:"players"`
}

type PlayerState struct {
	Info   connect.LobbyPlayer `json:"info"`
	Scores PlayerScores        `json:"scores"`
	Dice   []*Dice             `json:"dice"`
}

type PlayerScores struct {
	Total    int `json:"total"`
	Turn     int `json:"turn"`
	Selected int `json:"selected"`
}
