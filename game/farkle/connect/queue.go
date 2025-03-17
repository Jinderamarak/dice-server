package connect

import "github.com/google/uuid"

const CreateLobbyQueue = "/create/farkle"

func JoinLobbyQueue(gameId uuid.UUID) string {
	return "/join/farkle/" + gameId.String()
}
