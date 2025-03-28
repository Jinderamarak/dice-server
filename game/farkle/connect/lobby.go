package connect

import "github.com/google/uuid"

type CreateLobbyMessage struct {
	GameID uuid.UUID   `json:"gameId"`
	Target int         `json:"score"`
	Player LobbyPlayer `json:"player"`
}

type AcceptedLobbyMessage struct {
	GameID    uuid.UUID `json:"gameId"`
	ServerID  uuid.UUID `json:"serverId"`
	ServerURL string    `json:"serverUrl"`
}

type JoinLobbyMessage struct {
	Player LobbyPlayer `json:"player"`
}

type LobbyPlayer struct {
	UserID   uuid.UUID   `json:"userId"`
	Username string      `json:"username"`
	DiceSet  []LobbyDice `json:"diceSet"`
}

type LobbyDice struct {
	ID uuid.UUID `json:"id"`
}
