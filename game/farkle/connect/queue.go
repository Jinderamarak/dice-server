package connect

import (
	"dice-server/common/queue"
	"github.com/google/uuid"
)

var CreateLobbyQueue = queue.Declaration{
	Name:      "/farkle/create",
	Temporary: false,
	AutoAck:   false,
	QoS:       true,
}

func AcceptLobbyQueue(gameID uuid.UUID) queue.Declaration {
	return queue.Declaration{
		Name:      "/farkle/accept/" + gameID.String(),
		Temporary: true,
		AutoAck:   true,
		QoS:       false,
	}
}

func JoinLobbyQueue(gameID uuid.UUID) queue.Declaration {
	return queue.Declaration{
		Name:      "/farkle/join/" + gameID.String(),
		Temporary: true,
		AutoAck:   true,
		QoS:       false,
	}
}

func JoinedLobbyQueue(gameID uuid.UUID) queue.Declaration {
	return queue.Declaration{
		Name:      "/farkle/joined/" + gameID.String(),
		Temporary: true,
		AutoAck:   true,
		QoS:       false,
	}
}
