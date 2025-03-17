package data

import (
	"github.com/google/uuid"
)

type GameState struct {
	Id            uuid.UUID     `json:"id"`
	Target        int           `json:"target"`
	CurrentPlayer uuid.UUID     `json:"currentPlayer"`
	Players       []PlayerState `json:"players"`
}

type PlayerState struct {
	Info   LobbyPlayer  `json:"info"`
	Scores PlayerScores `json:"scores"`
	Dice   []Dice       `json:"dice"`
}

type PlayerScores struct {
	Total    int  `json:"total"`
	Turn     bool `json:"turn"`
	Selected int  `json:"selected"`
}
