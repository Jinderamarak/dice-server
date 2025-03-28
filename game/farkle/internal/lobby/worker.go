package lobby

import (
	"dice-server/common/queue"
	"dice-server/game/common/client"
	"dice-server/game/farkle/connect"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

func RunWorker(fail chan<- error, pool *queue.Pool, manager *client.WebSocketManager) {
	res := workerLoop(pool, manager)
	fail <- res
}

func workerLoop(pool *queue.Pool, manager *client.WebSocketManager) error {
	consumer := pool.GetConsumer(connect.CreateLobbyQueue)
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
	var createLobby connect.CreateLobbyMessage
	if err := json.Unmarshal(msg.Body, &createLobby); err != nil {
		return errors.Wrap(err, "failed to unmarshal create lobby message")
	}

	go startLobby(pool, manager, &createLobby)
	return nil
}

func startLobby(pool *queue.Pool, manager *client.WebSocketManager, msg *connect.CreateLobbyMessage) {
	//	TODO: Fill actual data
	acceptation := connect.AcceptedLobbyMessage{
		GameID:    msg.GameID,
		ServerID:  uuid.New(),
		ServerURL: "ws://localhost:9091",
	}

	publisher := pool.GetPublisher(connect.AcceptLobbyQueue(msg.GameID))
	_ = publisher.PublishJSON(acceptation)

	err := runLobby(pool, manager, msg)
	if err != nil {
		log.Println("Failed to run lobby:", err)
	}
}
