package connect

import (
	"github.com/google/uuid"
)

func PlayerToGameQueue(gameID, playerID uuid.UUID) string {
	return "/game/" + gameID.String() + "/from/" + playerID.String()
}

func GameToPlayerQueue(gameID, playerID uuid.UUID) string {
	return "/game/" + gameID.String() + "/to/" + playerID.String()
}
