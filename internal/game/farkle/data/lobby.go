package data

import "github.com/google/uuid"

const (
	CreateLobbyQueue = "/create/farkle"
	JoinLobbyQueue   = "/join/farkle/%s"
)

type CreateLobbyMessage struct {
	GameId uuid.UUID   `json:"gameId"`
	Target int         `json:"score"`
	Player LobbyPlayer `json:"player"`
}

type JoinLobbyMessage struct {
	Player LobbyPlayer `json:"player"`
}

type LobbyPlayer struct {
	UserId   uuid.UUID   `json:"userId"`
	Username string      `json:"username"`
	DiceSet  []LobbyDice `json:"diceSet"`
}

type LobbyDice struct {
	Id uuid.UUID `json:"id"`
}
