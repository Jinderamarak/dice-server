package connect

import "github.com/google/uuid"

const CreateLobbyQueue = "/create/farkle"

func JoinLobbyQueue(gameID uuid.UUID) string {
	return "/join/farkle/" + gameID.String()
}
