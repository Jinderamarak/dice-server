package lobby

import (
	"dice-server/internal/channel"
	"dice-server/internal/channel/message"
	"dice-server/internal/game/farkle"
	"dice-server/internal/game/farkle/data"
	"dice-server/internal/portal"
	"dice-server/internal/utility"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
)

const joinLobbyTimeout = time.Minute

func RunLobby(conn *amqp.Connection, msg *data.CreateLobbyMessage) {
	firstPlayer := msg.Player
	firstClient, err := createPlayer(conn, msg.GameId, firstPlayer.UserId)
	if err != nil {
		log.Println("Failed to create player:", err)
		return
	}

	secondClient, secondPlayer, err := waitForOtherPlayer(conn, msg.GameId)
	if err != nil {
		log.Println("Failed waiting for other player:", err)
		return
	}

	firstConnected := make(chan error)
	secondConnected := make(chan error)
	canceled := make(chan struct{})

	go waitForConnection(firstClient, firstPlayer.UserId, firstConnected, canceled)
	go waitForConnection(secondClient, secondPlayer.UserId, secondConnected, canceled)

	firstIsDone := false
	secondIsDone := false
	for {
		select {
		case first := <-firstConnected:
			if first != nil {
				log.Println("First player failed to connect:", first)
				close(canceled)
				abandonConnecting(firstClient, secondClient, "player failed to connect")
				return
			}

			firstIsDone = true
		case second := <-secondConnected:
			if second != nil {
				log.Println("Second player failed to connect:", second)
				close(canceled)
				abandonConnecting(firstClient, secondClient, "player failed to connect")
				return
			}

			secondIsDone = true
		case <-time.After(joinLobbyTimeout):
			close(canceled)
			abandonConnecting(firstClient, secondClient, "player did not connect")
			return
		}

		if firstIsDone && secondIsDone {
			break
		}
	}

	state := createGameState(&firstPlayer, secondPlayer, msg)
	go farkle.PlayFarkle(state, []*data.PlayerClient{firstClient, secondClient})
}

func abandonConnecting(first *data.PlayerClient, second *data.PlayerClient, reason string) {
	terminate := message.CraftControlTerminate(reason)

	_ = first.SendMessage(terminate)
	first.Close()

	_ = first.SendMessage(terminate)
	first.Close()
}

func createPlayer(conn *amqp.Connection, gameId, playerId uuid.UUID) (*data.PlayerClient, error) {
	writingQueue := portal.GameToPlayerQueue(gameId, playerId)
	readingQueue := portal.PlayerToGameQueue(gameId, playerId)

	rabbit, err := channel.OpenRabbitChannel(conn, writingQueue, readingQueue)
	if err != nil {
		return nil, err
	}

	return data.NewPlayerClient(playerId, rabbit), nil
}

func waitForOtherPlayer(conn *amqp.Connection, gameId uuid.UUID) (*data.PlayerClient, *data.LobbyPlayer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, nil, err
	}
	defer utility.CloseAndIgnore(ch)

	q, err := ch.QueueDeclare(
		fmt.Sprintf(data.JoinLobbyQueue, gameId),
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	messages, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	log.Println("Waiting for other player to join")
	for msg := range messages {
		var joinLobby data.JoinLobbyMessage
		err = json.Unmarshal(msg.Body, &joinLobby)
		if err != nil {
			log.Println("Join lobby attempt failed:", err)
			continue
		}

		client, err := createPlayer(conn, gameId, joinLobby.Player.UserId)
		if err != nil {
			return nil, nil, err
		}

		return client, &joinLobby.Player, nil
	}

	return nil, nil, errors.New("lobby ran out of messages")
}

func createGameState(first, second *data.LobbyPlayer, create *data.CreateLobbyMessage) *data.GameState {
	return &data.GameState{
		Id:            create.GameId,
		Target:        create.Target,
		CurrentPlayer: first.UserId,
		Players: []data.PlayerState{
			{
				Info:   *first,
				Scores: data.PlayerScores{},
				Dice:   nil,
			},
			{
				Info:   *first,
				Scores: data.PlayerScores{},
				Dice:   nil,
			},
		},
	}
}

func waitForConnection(client *data.PlayerClient, playerId uuid.UUID, connected chan<- error, canceled <-chan struct{}) {
	client.SetOnTurn()
	defer client.SetOffTurn()

	if client.GetState() == data.PlayerStateConnected {
		close(connected)
		return
	}

	for {
		select {
		case msg := <-client.ReadChannel():
			if msg.Variant == message.VarControlConnected {
				var controlConnected message.VariantControlConnected
				err := msg.UnmarshalData(&controlConnected)
				if err != nil {
					continue
				}

				if controlConnected.UserId == playerId {
					close(connected)
					return
				}
			}
		case <-client.Closed():
			connected <- errors.New("client closed")
			return
		case <-canceled:
			return
		}
	}
}
