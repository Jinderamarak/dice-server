package loop

import (
	"dice-server/common/auth/token"
	"dice-server/common/channel"
	"dice-server/common/channel/message"
	wschan "dice-server/portal/internal/channel"
	"log"
)

func OpenPortal(player *wschan.WebSocketChannel, server *channel.RabbitChannel, gameToken *token.GameToken) {
	defer player.Close()
	defer server.Close()

	if err := server.SendMessage(message.CraftControlConnected(gameToken.UserId)); err != nil {
		log.Println("Failed to send connected message to rabbit")
		_ = player.WriteMessage(message.CraftControlError("control-internal", "internal server error"))
		return
	}

	for {
		select {
		case msg := <-player.ReadChannel():
			if err := message.ValidateUserMessage(msg); err != nil {
				log.Println("Invalid message from websocket:", err)
				_ = player.WriteMessage(message.CraftControlError("control-validate", "sent control message"))
				continue
			}

			if err := server.SendMessage(msg); err != nil {
				log.Println("Failed to send message to rabbit:", err)
				_ = player.WriteMessage(message.CraftControlError("control-internal", "internal server error"))
				return
			}
		case msg := <-server.ReadChannel():
			if err := player.WriteMessage(msg); err != nil {
				log.Println("Failed to send message to websocket:", err)
				_ = server.SendMessage(message.CraftControlDisconnected(gameToken.UserId))
				return
			}
		case <-player.Closed():
			log.Println("Closed by websocket")
			_ = server.SendMessage(message.CraftControlDisconnected(gameToken.UserId))
			return
		case <-server.Closed():
			log.Println("Closed by rabbit")
			_ = player.WriteMessage(message.CraftControlError("control-internal", "internal server error"))
			return
		}
	}
}
