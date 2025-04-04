package lobby

import (
	"dice-server/common/auth/token"
	"dice-server/common/queue"
	"dice-server/game/common/client"
	"dice-server/game/farkle/connect"
	"dice-server/game/farkle/internal/config"
	"encoding/json"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
)

func RunWorker(fail chan<- error, pool *queue.Pool, manager *client.WebSocketManager) {
	res := workerLoop(pool, manager)
	fail <- res
}

func workerLoop(pool *queue.Pool, manager *client.WebSocketManager) error {
	consumer := pool.GetConsumer(connect.CreateLobbyQueue)
	defer consumer.Close()

	messages := consumer.Consume()
	for msg := range messages {
		err := attemptLobby(pool, manager, &msg)
		if err != nil {
			if err = msg.Nack(false, false); err != nil {
				return errors.Wrap(err, "failed to nack message")
			}
		} else {
			if err = msg.Ack(false); err != nil {
				return errors.Wrap(err, "failed to ack message")
			}
		}
	}

	return errors.New("worker consumer ended")
}

func attemptLobby(pool *queue.Pool, manager *client.WebSocketManager, msg *amqp.Delivery) error {
	var createLobby connect.CreateFarkleRequest
	if err := json.Unmarshal(msg.Body, &createLobby); err != nil {
		return errors.Wrap(err, "failed to unmarshal create lobby message")
	}

	go startLobby(pool, manager, &createLobby)
	return nil
}

func startLobby(pool *queue.Pool, manager *client.WebSocketManager, msg *connect.CreateFarkleRequest) {

	auth := token.NewGameToken(msg.Player.UserID, msg.GameID, config.Config.Server.ID, config.Config.Server.Host, config.Config.Auth.Issuer, time.Now())
	authToken, err := auth.Sign([]byte(config.Config.Auth.Secret))
	if err != nil {
		log.Println("Failed to sign auth token:", err)
		return
	}

	publisher := pool.GetPublisher(connect.AcceptLobbyQueue(msg.GameID))
	err = publisher.PublishJSON(connect.CreateFarkleResponse{
		GameID:     msg.GameID,
		ServerID:   config.Config.Server.ID,
		ServerHost: config.Config.Server.Host,
		Auth:       authToken,
	})
	if err != nil {
		log.Println("Failed to publish acceptation message:", err)
		return
	}
	publisher.Close()

	err = runLobby(pool, manager, msg)
	if err != nil {
		log.Println("Failed to run lobby:", err)
	}
}
