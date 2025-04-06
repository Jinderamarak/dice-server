package lobby

import (
	"dice-server/common/auth/token"
	"dice-server/common/queue"
	"dice-server/game/common/client"
	"dice-server/game/farkle/connect"
	"dice-server/game/farkle/internal/config"
	"dice-server/game/farkle/internal/data"
	"dice-server/game/farkle/internal/logic"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"log"
	"time"
)

const (
	playerJoinTimeout    = time.Minute * 5
	playersReadyTimeout  = time.Minute
	playerPleaseInterval = time.Second * 5
)

func runLobby(pool *queue.Pool, manager *client.WebSocketManager, msg *connect.CreateFarkleRequest) error {
	firstPlayer := msg.Player
	firstClient, err := createPlayer(manager, msg.GameID, firstPlayer.UserID)
	if err != nil {
		return errors.Wrap(err, "failed to create first player")
	}

	secondClient, secondPlayer, err := waitForOtherPlayer(pool, manager, msg.GameID, time.Now().Add(playerJoinTimeout))
	if err != nil {
		abandonConnecting(manager, msg.GameID, firstClient, nil, "failed to wait for other player")
		return errors.Wrap(err, "failed to wait for other player")
	}

	if err := firstClient.Send(data.CraftPlayerJoining(secondPlayer)); err != nil {
		abandonConnecting(manager, msg.GameID, firstClient, secondClient, "failed to send player joining")
		return errors.Wrap(err, "failed to send player joining")
	}

	firstConnected := make(chan error)
	secondConnected := make(chan error)
	canceled := make(chan struct{})

	log.Println("Waiting for both players to connect")
	go waitForReady(firstClient, firstPlayer.UserID, firstConnected, canceled)
	go waitForReady(secondClient, secondPlayer.UserID, secondConnected, canceled)

	firstIsDone := false
	secondIsDone := false
	deadline := time.Now().Add(playersReadyTimeout)
	for !firstIsDone || !secondIsDone {
		select {
		case first, ok := <-firstConnected:
			if ok && first != nil {
				close(canceled)
				abandonConnecting(manager, msg.GameID, firstClient, secondClient, "player failed to connect")
				return errors.Wrap(first, "first player failed to connect")
			}

			if !firstIsDone {
				log.Println("First player just connected")
			}
			firstIsDone = true
		case second, ok := <-secondConnected:
			if ok && second != nil {
				close(canceled)
				abandonConnecting(manager, msg.GameID, firstClient, secondClient, "player failed to connect")
				return errors.Wrap(second, "second player failed to connect")
			}

			if !secondIsDone {
				log.Println("Second player just connected")
			}
			secondIsDone = true
		case <-time.After(time.Until(deadline)):
			close(canceled)
			abandonConnecting(manager, msg.GameID, firstClient, secondClient, "player did not connect")
			return errors.Wrap(err, "players took too long to connect")
		}
	}

	firstClient.DrainMessages()
	secondClient.DrainMessages()

	game := logic.NewFarkleGame(manager, firstClient, secondClient, &firstPlayer, secondPlayer, msg)
	game.Play()

	return nil
}

func abandonConnecting(manager *client.WebSocketManager, gameID uuid.UUID, first *data.PlayerClient, second *data.PlayerClient, reason string) {
	terminate := data.CraftTerminate(reason)
	if first != nil {
		_ = first.Send(terminate)
	}
	if second != nil {
		_ = second.Send(terminate)
	}

	time.After(time.Second)
	manager.CloseGame(gameID)
}

func createPlayer(manager *client.WebSocketManager, gameID, userID uuid.UUID) (*data.PlayerClient, error) {
	wsClient := manager.GetClient(gameID, userID)
	player := data.NewPlayerClient(wsClient)
	player.SetTurn(true)
	return player, nil
}

func waitForOtherPlayer(pool *queue.Pool, manager *client.WebSocketManager, gameID uuid.UUID, deadline time.Time) (*data.PlayerClient, *connect.LobbyPlayer, error) {
	log.Println("Waiting for other player to join:", gameID)

	consumer := pool.GetConsumer(connect.JoinLobbyQueue(gameID))
	defer consumer.Close()

	messages := consumer.Consume()
	for {
		select {
		case <-time.After(time.Until(deadline)):
			return nil, nil, errors.New("lobby ran out of time")
		case msg, ok := <-messages:
			if !ok {
				return nil, nil, errors.New("lobby ran out of messages")
			}

			var joinLobby connect.JoinFarkleRequest
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

			auth := token.NewGameToken(joinLobby.Player.UserID, gameID, config.Config.Server.ID, config.Config.Server.Host, config.Config.Auth.Issuer, time.Now())
			authToken, err := auth.Sign([]byte(config.Config.Auth.Secret))
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to sign auth token")
			}

			publisher := pool.GetPublisher(connect.JoinedLobbyQueue(gameID))
			err = publisher.PublishJSON(connect.JoinFarkleResponse{
				GameID:     gameID,
				ServerID:   config.Config.Server.ID,
				ServerHost: config.Config.Server.Host,
				Auth:       authToken,
			})

			publisher.Close()
			if err != nil {
				return nil, nil, err
			}

			return c, &joinLobby.Player, nil
		}
	}
}

func waitForReady(client *data.PlayerClient, playerID uuid.UUID, connected chan<- error, canceled <-chan struct{}) {
	defer client.SetTurn(false)

	for {
		_ = client.Send(data.CraftPleaseReady())
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
		case <-time.After(playerPleaseInterval):
			continue
		case <-client.Closing():
			connected <- errors.New("client closed")
			return
		case <-canceled:
			return
		}
	}
}
