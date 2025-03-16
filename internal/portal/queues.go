package portal

import (
	"fmt"
	"github.com/google/uuid"
)

func PlayerToGameQueue(gameId uuid.UUID, playerId uuid.UUID) string {
	return fmt.Sprintf("/game/%s/from/%s", gameId, playerId)
}

func GameToPlayerQueue(gameId uuid.UUID, playerId uuid.UUID) string {
	return fmt.Sprintf("/game/%s/to/%s", gameId, playerId)
}
