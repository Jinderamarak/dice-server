package connect

import (
	"github.com/google/uuid"
)

func PlayerToGameQueue(gameId, playerId uuid.UUID) string {
	return "/game/" + gameId.String() + "/from/" + playerId.String()
}

func GameToPlayerQueue(gameId, playerId uuid.UUID) string {
	return "/game/" + gameId.String() + "/to/" + playerId.String()
}
