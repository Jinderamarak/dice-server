package farkle

import (
	"dice-server/internal/channel/message"
	"dice-server/internal/game/farkle/data"
)

func PlayFarkle(state *data.GameState, clients []*data.PlayerClient) {
	defer func() {
		for _, client := range clients {
			client.Close()
		}
	}()

	broadcast(clients, data.CraftGameBegin(*state))
}

func broadcast(clients []*data.PlayerClient, msg *message.Message) {
	for _, client := range clients {
		_ = client.SendMessage(msg)
	}
}
