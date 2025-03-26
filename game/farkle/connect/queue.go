package connect

import (
	"dice-server/common/queue"
	"github.com/google/uuid"
)

const CreateLobbyQueue = "/create/farkle"

var CreateLobbyQueueDeclaration = queue.Declaration{
	Name:      "/create/farkle",
	Temporary: false,
	AutoAck:   false,
	QoS:       true,
}

func JoinLobbyQueue(gameID uuid.UUID) string {
	return "/join/farkle/" + gameID.String()
}

func JoinLobbyQueueDeclaration(gameID uuid.UUID) queue.Declaration {
	return queue.Declaration{
		Name:      JoinLobbyQueue(gameID),
		Temporary: true,
		AutoAck:   false,
		QoS:       false,
	}
}
