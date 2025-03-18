package lobby

import (
	"dice-server/common/channel"
	"dice-server/common/channel/message"
	"dice-server/common/utility"
	"dice-server/game/farkle/connect"
	"dice-server/game/farkle/internal/data"
	"dice-server/game/farkle/internal/logic"
	portal "dice-server/portal/connect"
	"encoding/json"
	"errors"
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

	log.Println("Waiting for both players to connect")
	go waitForConnection(firstClient, firstPlayer.UserId, firstConnected, canceled)
	go waitForConnection(secondClient, secondPlayer.UserId, secondConnected, canceled)

	firstIsDone := false
	secondIsDone := false
	deadline := time.Now().Add(joinLobbyTimeout)
	for {
		select {
		case first, ok := <-firstConnected:
			if ok && first != nil {
				log.Println("First player failed to connect:", first)
				close(canceled)
				abandonConnecting(firstClient, secondClient, "player failed to connect")
				return
			}

			if !firstIsDone {
				log.Println("First player just connected")
			}
			firstIsDone = true
		case second, ok := <-secondConnected:
			if ok && second != nil {
				log.Println("Second player failed to connect:", second)
				close(canceled)
				abandonConnecting(firstClient, secondClient, "player failed to connect")
				return
			}

			if !secondIsDone {
				log.Println("Second player just connected")
			}
			secondIsDone = true
		case <-time.After(deadline.Sub(time.Now())):
			log.Println("Players took too long")
			close(canceled)
			abandonConnecting(firstClient, secondClient, "player did not connect")
			return
		}

		if firstIsDone && secondIsDone {
			break
		}
	}

	log.Println("Creating game of farkle")
	state := createGameState(&firstPlayer, secondPlayer, msg)
	go logic.PlayFarkle(state, []*data.PlayerClient{firstClient, secondClient})
}

func abandonConnecting(first *data.PlayerClient, second *data.PlayerClient, reason string) {
	terminate := message.CraftControlTerminate(reason)

	_ = first.SendMessage(terminate)
	first.Close()

	_ = second.SendMessage(terminate)
	second.Close()
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
		connect.JoinLobbyQueue(gameId),
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
	firstDice := make([]*data.Dice, len(first.DiceSet))
	for i, d := range first.DiceSet {
		firstDice[i] = data.NewDice(d.Id)
	}

	secondDice := make([]*data.Dice, len(second.DiceSet))
	for i, d := range second.DiceSet {
		secondDice[i] = data.NewDice(d.Id)
	}

	return &data.GameState{
		Id:            create.GameId,
		Target:        create.Target,
		CurrentPlayer: first.UserId,
		Players: []*data.PlayerState{
			{
				Info:   *first,
				Scores: data.PlayerScores{},
				Dice:   firstDice,
			},
			{
				Info:   *second,
				Scores: data.PlayerScores{},
				Dice:   secondDice,
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
