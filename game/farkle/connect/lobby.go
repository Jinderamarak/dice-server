package connect

import (
	"github.com/google/uuid"
)

type CreateFarkleRequest struct {
	GameID uuid.UUID   `json:"gameId"`
	Target int         `json:"score"`
	Player LobbyPlayer `json:"player"`
}

type CreateFarkleResponse struct {
	GameID     uuid.UUID `json:"gameId"`
	ServerID   uuid.UUID `json:"serverId"`
	ServerHost string    `json:"serverHost"`
	Auth       string    `json:"auth"`
}

type JoinFarkleRequest struct {
	Player LobbyPlayer `json:"player"`
}

type JoinFarkleResponse struct {
	GameID     uuid.UUID `json:"gameId"`
	ServerID   uuid.UUID `json:"serverId"`
	ServerHost string    `json:"serverHost"`
	Auth       string    `json:"auth"`
}

type LobbyPlayer struct {
	UserID   uuid.UUID   `json:"userId"`
	Username string      `json:"username"`
	DiceSet  []LobbyDice `json:"diceSet"`
}
