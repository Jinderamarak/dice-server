package logic

import (
	"dice-server/game/common/client"
	"dice-server/game/farkle/connect"
	"dice-server/game/farkle/internal/data"
	"errors"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	pickTime       = time.Minute
	terminateSleep = time.Second * 10
)

const (
	errGeneral      = "farkle-server"
	errBadData      = "farkle-bad-data"
	errBadDice      = "farkle-bad-dice"
	errExtraDice    = "farkle-extra-dice"
	errNoneSelected = "farkle-none-selected"
	errBadPlayer    = "farkle-bad-player"
	errUnexpected   = "farkle-unexpected"
)

type FarkleGame struct {
	manager  *client.WebSocketManager
	clients  []*data.PlayerClient
	timeouts []int

	stateMu sync.RWMutex
	state   *data.GameState
}

func NewFarkleGame(
	manager *client.WebSocketManager,
	firstClient, secondClient *data.PlayerClient,
	firstPlayer, secondPlayer *connect.LobbyPlayer,
	create *connect.CreateFarkleRequest,
) *FarkleGame {
	firstDice := make([]*data.Dice, len(firstPlayer.DiceSet))
	for i, d := range firstPlayer.DiceSet {
		firstDice[i] = data.NewDice(d.ID)
	}

	secondDice := make([]*data.Dice, len(secondPlayer.DiceSet))
	for i, d := range secondPlayer.DiceSet {
		secondDice[i] = data.NewDice(d.ID)
	}

	return &FarkleGame{
		manager:  manager,
		clients:  []*data.PlayerClient{firstClient, secondClient},
		timeouts: []int{0, 0},
		state: &data.GameState{
			ID:            create.GameID,
			Target:        create.Target,
			PickSeconds:   uint(math.Round(pickTime.Seconds())),
			CurrentPlayer: firstPlayer.UserID,
			Players: []*data.PlayerState{
				{
					Info:   *firstPlayer,
					Scores: data.PlayerScores{},
					Dice:   firstDice,
				},
				{
					Info:   *secondPlayer,
					Scores: data.PlayerScores{},
					Dice:   secondDice,
				},
			},
		},
	}
}

func (game *FarkleGame) Play() {
	defer game.manager.CloseGame(game.state.ID)

	for _, c := range game.clients {
		c.SetGameStateHandler(func() *data.GameState {
			game.stateMu.RLock()
			defer game.stateMu.RUnlock()
			return game.state
		})
	}

	game.broadcast(data.CraftGameBegin(game.state))

	err := game.gameLoop()
	if err != nil {
		log.Println("Game terminated:", err)
		game.broadcast(data.CraftError(errGeneral, "server error"))
	}

	game.broadcast(data.CraftTerminate("game closed"))
	time.Sleep(terminateSleep)
}

func (game *FarkleGame) broadcast(msg *client.Message) {
	for _, c := range game.clients {
		err := c.Send(msg)
		if err != nil {
			log.Println("Error broadcasting to client:", err)
		}
	}
}

func (game *FarkleGame) gameLoop() error {
	currentPlayerIdx := rand.Intn(len(game.state.Players))
	for {
		currentPlayerIdx = (currentPlayerIdx + 1) % len(game.state.Players)
		currentPlayer := game.state.Players[currentPlayerIdx]
		currentClient := game.clients[currentPlayerIdx]
		currentTimeouts := &game.timeouts[currentPlayerIdx]

		game.stateMu.Lock()
		game.state.CurrentPlayer = currentPlayer.Info.UserID
		game.stateMu.Unlock()

		err := game.turnLoop(currentPlayer, currentClient, currentTimeouts)
		if err != nil {
			return err
		}

		if *currentTimeouts >= 2 {
			log.Println("Player timed out, game ended")
			otherPlayerIdx := (currentPlayerIdx + 1) % len(game.state.Players)
			otherPlayer := game.state.Players[otherPlayerIdx]
			game.broadcast(data.CraftGameEnd(otherPlayer.Info.UserID))
			return nil
		}

		game.broadcast(data.CraftUpdateScore(currentPlayer.Info.UserID, currentPlayer.Scores))

		if currentPlayer.Scores.Total >= game.state.Target {
			log.Println("Game ended, winner:", currentPlayer.Info.Username)
			game.broadcast(data.CraftGameEnd(currentPlayer.Info.UserID))
			return nil
		}
	}
}

func (game *FarkleGame) turnLoop(playerState *data.PlayerState, playerClient *data.PlayerClient, timeouts *int) error {
	log.Println("New turn:", playerState.Info.Username)

	playerClient.SetTurn(true)
	defer playerClient.SetTurn(false)

	game.broadcast(data.CraftTurnBegin(playerState.Info.UserID))

	game.stateMu.Lock()
	playerState.Scores.Turn = 0
	playerState.Scores.Selected = 0
	resetDice(playerState.Dice)
	game.stateMu.Unlock()

	for {
		game.stateMu.Lock()
		rollDice(playerState.Dice)
		game.stateMu.Unlock()

		busted := hasBusted(countValues(playerState.Dice, true))
		game.broadcast(data.CraftDiceRoll(playerState.Dice, busted))

		if busted {
			log.Println("Player busted")

			game.stateMu.Lock()
			playerState.Scores.Turn = 0
			playerState.Scores.Selected = 0
			game.stateMu.Unlock()

			game.broadcast(data.CraftUpdateScore(playerState.Info.UserID, playerState.Scores))
			return nil
		}

		rollAgain, err := game.diceSelection(
			playerState,
			playerClient,
			time.Now().Add(time.Duration(game.state.PickSeconds)*time.Second),
		)

		if err != nil {
			if errors.Is(err, client.ErrRecvTimeout) {
				game.stateMu.Lock()
				playerState.Scores.Selected = 0
				playerState.Scores.Turn = 0
				game.stateMu.Unlock()

				game.broadcast(data.CraftTurnTimeout(playerState.Info.UserID))

				*timeouts += 1
				return nil
			} else {
				return err
			}
		}

		game.stateMu.Lock()
		playerState.Scores.Turn += playerState.Scores.Selected
		playerState.Scores.Selected = 0
		moveSelectedToUnplayable(playerState.Dice)

		if !rollAgain {
			playerState.Scores.Total += playerState.Scores.Turn
			game.stateMu.Unlock()
			return nil
		}
		game.stateMu.Unlock()

		game.broadcast(data.CraftUpdateScore(playerState.Info.UserID, playerState.Scores))
	}
}

func (game *FarkleGame) diceSelection(playerState *data.PlayerState, playerClient *data.PlayerClient, deadline time.Time) (bool, error) {
	hasExtraDice := false
	for {
		log.Println("Waiting for next step")
		step, err := playerClient.Receive(deadline.Sub(time.Now()))
		if err != nil {
			return false, err
		}
		log.Println("Next step is:", step.Variant)

		switch step.Variant {
		case data.VarDiceTouch:
			var diceTouch data.VariantDiceTouch
			if err = step.UnmarshalData(&diceTouch); err != nil {
				log.Println("Error unmarshalling dice touch:", err)
				_ = playerClient.Send(data.CraftError(errBadData, "sent bad data"))
				continue
			}

			game.stateMu.Lock()
			validDice := touchDice(playerState.Dice, &diceTouch)
			game.stateMu.Unlock()

			if !validDice {
				log.Println("Player touched bad dice")
				_ = playerClient.Send(data.CraftError(errBadDice, "touched bad dice"))
				continue
			}

			game.broadcast(data.CraftDiceTouched(playerState.Info.UserID, playerState.Dice))

			game.stateMu.Lock()
			selected, extra := scoreCounts(countValues(playerState.Dice, false))
			playerState.Scores.Selected = selected
			if extra {
				playerState.Scores.Selected = 0
			}
			game.stateMu.Unlock()

			hasExtraDice = extra
			game.broadcast(data.CraftUpdateScore(playerState.Info.UserID, playerState.Scores))

		case data.VarScoreRoll:
			var scoreRoll data.VariantScoreRoll
			if err = step.UnmarshalData(&scoreRoll); err != nil {
				log.Println("Error unmarshalling score and roll:", err)
				_ = playerClient.Send(data.CraftError(errBadData, "sent bad data"))
				continue
			}

			if playerState.Info.UserID != scoreRoll.PlayerID {
				_ = playerClient.Send(data.CraftError(errBadPlayer, "bad player id"))
				continue
			}

			if playerState.Scores.Selected == 0 {
				_ = playerClient.Send(data.CraftError(errNoneSelected, "no dice selected"))
				continue
			}

			if hasExtraDice {
				_ = playerClient.Send(data.CraftError(errExtraDice, "extra dice selected"))
				continue
			}

			game.broadcast(data.CraftScoreRoll(playerState.Info.UserID))
			return true, nil

		case data.VarEndTurn:
			var endTurn data.VariantEndTurn
			if err = step.UnmarshalData(&endTurn); err != nil {
				log.Println("Error unmarshalling end turn:", err)
				_ = playerClient.Send(data.CraftError(errBadData, "sent bad data"))
				continue
			}

			if playerState.Info.UserID != endTurn.PlayerID {
				_ = playerClient.Send(data.CraftError(errBadPlayer, "bad player id"))
				continue
			}

			if hasExtraDice {
				_ = playerClient.Send(data.CraftError(errExtraDice, "extra dice selected"))
				continue
			}

			game.broadcast(data.CraftEndTurn(playerState.Info.UserID))
			return false, nil
		default:
			_ = playerClient.Send(data.CraftError(errUnexpected, "unexpected variant"))
		}
	}
}

func resetDice(dice []*data.Dice) {
	for _, d := range dice {
		d.Selected = false
		d.Playable = true
	}
}

func rollDice(dice []*data.Dice) {
	hasPlayableDice := false
	for _, d := range dice {
		if d.Playable {
			hasPlayableDice = true
			break
		}
	}

	for _, d := range dice {
		d.Playable = d.Playable || !hasPlayableDice
		d.Roll()
	}
}

func touchDice(dice []*data.Dice, touch *data.VariantDiceTouch) bool {
	for _, d := range dice {
		if d.ID == touch.DiceID {
			if !d.Playable {
				return false
			}
			d.Selected = touch.Selected
			return true
		}
	}
	return false
}

func moveSelectedToUnplayable(dice []*data.Dice) {
	for _, d := range dice {
		if d.Selected {
			d.Playable = false
		}
		d.Selected = false
	}
}
