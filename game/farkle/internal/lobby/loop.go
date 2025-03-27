package lobby

import (
	"dice-server/common/queue"
	"dice-server/game/common/client"
	"dice-server/game/farkle/connect"
	"dice-server/game/farkle/internal/data"
	"dice-server/game/farkle/internal/logic"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"log"
	"time"
)

const joinLobbyTimeout = time.Minute

func RunLobby(pool *queue.Pool, manager *client.WebSocketManager, msg *connect.CreateLobbyMessage) {
	firstPlayer := msg.Player
	firstClient, err := createPlayer(manager, msg.GameID, firstPlayer.UserID)
	if err != nil {
		log.Println("Failed to create player:", err)
		return
	}

	secondClient, secondPlayer, err := waitForOtherPlayer(pool, manager, msg.GameID)
	if err != nil {
		log.Println("Failed waiting for other player:", err)
		return
	}

	firstConnected := make(chan error)
	secondConnected := make(chan error)
	canceled := make(chan struct{})

	log.Println("Waiting for both players to connect")
	go waitForReady(firstClient, firstPlayer.UserID, firstConnected, canceled)
	go waitForReady(secondClient, secondPlayer.UserID, secondConnected, canceled)

	firstIsDone := false
	secondIsDone := false
	deadline := time.Now().Add(joinLobbyTimeout)
	for !firstIsDone || !secondIsDone {
		select {
		case first, ok := <-firstConnected:
			if ok && first != nil {
				log.Println("First player failed to connect:", first)
				close(canceled)
				abandonConnecting(manager, msg.GameID, firstClient, secondClient, "player failed to connect")
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
				abandonConnecting(manager, msg.GameID, firstClient, secondClient, "player failed to connect")
				return
			}

			if !secondIsDone {
				log.Println("Second player just connected")
			}
			secondIsDone = true
		case <-time.After(deadline.Sub(time.Now())):
			log.Println("Players took too long")
			close(canceled)
			abandonConnecting(manager, msg.GameID, firstClient, secondClient, "player did not connect")
			return
		}
	}

	log.Println("Creating game of farkle")
	state := createGameState(&firstPlayer, secondPlayer, msg)
	go logic.PlayFarkle(manager, state, []*data.PlayerClient{firstClient, secondClient})
}

func abandonConnecting(manager *client.WebSocketManager, gameID uuid.UUID, first *data.PlayerClient, second *data.PlayerClient, reason string) {
	terminate := data.CraftTerminate(reason)
	_ = first.Send(terminate)
	_ = second.Send(terminate)

	time.After(time.Second)
	manager.CloseGame(gameID)
}

func createPlayer(manager *client.WebSocketManager, gameID, userID uuid.UUID) (*data.PlayerClient, error) {
	wsClient := manager.GetClient(gameID, userID)
	return data.NewPlayerClient(wsClient), nil
}

func waitForOtherPlayer(pool *queue.Pool, manager *client.WebSocketManager, gameID uuid.UUID) (*data.PlayerClient, *connect.LobbyPlayer, error) {
	log.Println("Waiting for other player to join:", gameID)

	consumer := pool.GetConsumer(connect.JoinLobbyQueueDeclaration(gameID))
	defer consumer.Close()

	messages := consumer.Consume()
	for msg := range messages {
		var joinLobby connect.JoinLobbyMessage
		err := json.Unmarshal(msg.Body, &joinLobby)
		if err != nil {
			log.Println("Join lobby attempt failed:", err)
			continue
		}

		log.Println("Joining player:", joinLobby.Player.Username)
		c, err := createPlayer(manager, gameID, joinLobby.Player.UserID)
		if err != nil {
			return nil, nil, err
		}

		return c, &joinLobby.Player, nil
	}

	return nil, nil, errors.New("lobby ran out of messages")
}

func createGameState(first, second *connect.LobbyPlayer, create *connect.CreateLobbyMessage) *data.GameState {
	firstDice := make([]*data.Dice, len(first.DiceSet))
	for i, d := range first.DiceSet {
		firstDice[i] = data.NewDice(d.ID)
	}

	secondDice := make([]*data.Dice, len(second.DiceSet))
	for i, d := range second.DiceSet {
		secondDice[i] = data.NewDice(d.ID)
	}

	return &data.GameState{
		ID:            create.GameID,
		Target:        create.Target,
		CurrentPlayer: first.UserID,
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

func waitForReady(client *data.PlayerClient, playerID uuid.UUID, connected chan<- error, canceled <-chan struct{}) {
	for {
		select {
		case msg := <-client.Receiving():
			if msg.Variant == data.VarPlayerReady {
				var playerReady data.VariantPlayerReady
				err := msg.UnmarshalData(&playerReady)
				if err != nil {
					log.Println("Failed to unmarshal player ready:", err)
					continue
				}

				if playerReady.PlayerID == playerID {
					close(connected)
					return
				}
			}
		case <-client.Closing():
			connected <- errors.New("client closed")
			return
		case <-canceled:
			return
		}
	}
}
