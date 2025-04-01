package connect

import (
	"github.com/google/uuid"
)

type CreateLobbyMessage struct {
	GameID uuid.UUID   `json:"gameId"`
	Target int         `json:"score"`
	Player LobbyPlayer `json:"player"`
}

type AcceptedLobbyMessage struct {
	GameID     uuid.UUID `json:"gameId"`
	ServerID   uuid.UUID `json:"serverId"`
	ServerHost string    `json:"serverHost"`
}

type JoinLobbyMessage struct {
	Player LobbyPlayer `json:"player"`
}

type JoinedLobbyMessage struct {
	UserID     uuid.UUID `json:"userId"`
	ServerID   uuid.UUID `json:"serverId"`
	ServerHost string    `json:"serverHost"`
}

type LobbyPlayer struct {
	UserID   uuid.UUID   `json:"userId"`
	Username string      `json:"username"`
	DiceSet  []LobbyDice `json:"diceSet"`
}

type LobbyDice struct {
	ID uuid.UUID `json:"id"`
}
