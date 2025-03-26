package logic

import (
	"dice-server/game/common/client"
	"dice-server/game/farkle/internal/data"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"
)

const (
	beginSleep     = time.Second * 3
	turnBeginSleep = time.Second
	pickTimeout    = time.Minute
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

func PlayFarkle(gameState *data.GameState, clients []*data.PlayerClient) {
	defer func() {
		for _, c := range clients {
			c.Close()
		}
	}()

	stateMu := sync.RWMutex{}
	for _, c := range clients {
		c.SetGameStateHandler(func() *data.GameState {
			stateMu.RLock()
			defer stateMu.RUnlock()
			return gameState
		})
	}

	broadcast(clients, data.CraftGameBegin(gameState))
	time.Sleep(beginSleep)

	err := gameLoop(gameState, &stateMu, clients)
	if err != nil {
		log.Println("Game terminated:", err)
		broadcast(clients, data.CraftError(errGeneral, "server error"))
	}

	broadcast(clients, data.CraftTerminate("game closed"))
	time.Sleep(terminateSleep)
}

func broadcast(clients []*data.PlayerClient, msg *client.Message) {
	for _, c := range clients {
		err := c.Send(msg)
		if err != nil {
			log.Println("Error broadcasting to client:", err)
		}
	}
}

func gameLoop(gameState *data.GameState, stateMu *sync.RWMutex, clients []*data.PlayerClient) error {
	currentPlayerIdx := rand.Intn(len(gameState.Players))
	for {
		currentPlayerIdx = (currentPlayerIdx + 1) % len(gameState.Players)
		currentPlayer := gameState.Players[currentPlayerIdx]
		currentClient := clients[currentPlayerIdx]

		stateMu.Lock()
		gameState.CurrentPlayer = currentPlayer.Info.UserID
		stateMu.Unlock()

		err := turnLoop(clients, stateMu, currentPlayer, currentClient)
		if err != nil {
			return err
		}

		broadcast(clients, data.CraftUpdateScore(currentPlayer.Info.UserID, currentPlayer.Scores))

		if currentPlayer.Scores.Total >= gameState.Target {
			log.Println("Game ended, winner:", currentPlayer.Info.Username)
			broadcast(clients, data.CraftGameEnd(currentPlayer.Info.UserID))
			return nil
		}
	}
}

func turnLoop(clients []*data.PlayerClient, stateMu *sync.RWMutex, playerState *data.PlayerState, playerClient *data.PlayerClient) error {
	log.Println("New turn:", playerState.Info.Username)

	playerClient.SetTurn(true)
	defer playerClient.SetTurn(false)

	broadcast(clients, data.CraftTurnBegin(playerState.Info.UserID))
	time.Sleep(turnBeginSleep)

	stateMu.Lock()
	playerState.Scores.Turn = 0
	playerState.Scores.Selected = 0
	resetDice(playerState.Dice)
	stateMu.Unlock()

	for {
		stateMu.Lock()
		rollDice(playerState.Dice)
		stateMu.Unlock()

		busted := hasBusted(countValues(playerState.Dice, true))
		broadcast(clients, data.CraftDiceRoll(playerState.Dice, busted))

		if busted {
			log.Println("Player busted")

			stateMu.Lock()
			playerState.Scores.Turn = 0
			playerState.Scores.Selected = 0
			stateMu.Unlock()

			broadcast(clients, data.CraftUpdateScore(playerState.Info.UserID, playerState.Scores))
			return nil
		}

		rollAgain, err := diceSelection(clients, stateMu, playerState, playerClient, time.Now().Add(pickTimeout))
		if err != nil {
			if errors.Is(err, client.ErrRecvTimeout) {
				stateMu.Lock()
				playerState.Scores.Selected = 0
				playerState.Scores.Turn = 0
				stateMu.Unlock()

				broadcast(clients, data.CraftTurnTimeout(playerState.Info.UserID))
				return nil
			} else {
				return err
			}
		}

		stateMu.Lock()
		playerState.Scores.Turn += playerState.Scores.Selected
		playerState.Scores.Selected = 0
		moveSelectedToUnplayable(playerState.Dice)

		if !rollAgain {
			playerState.Scores.Total += playerState.Scores.Turn
			stateMu.Unlock()
			return nil
		}
		stateMu.Unlock()

		broadcast(clients, data.CraftUpdateScore(playerState.Info.UserID, playerState.Scores))
	}
}

func diceSelection(clients []*data.PlayerClient, stateMu *sync.RWMutex, playerState *data.PlayerState, playerClient *data.PlayerClient, deadline time.Time) (bool, error) {
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

			stateMu.Lock()
			validDice := touchDice(playerState.Dice, &diceTouch)
			stateMu.Unlock()

			if !validDice {
				log.Println("Player touched bad dice")
				_ = playerClient.Send(data.CraftError(errBadDice, "touched bad dice"))
				continue
			}

			broadcast(clients, data.CraftDiceTouched(playerState.Info.UserID, playerState.Dice))

			stateMu.Lock()
			selected, extra := scoreCounts(countValues(playerState.Dice, false))
			playerState.Scores.Selected = selected
			if extra {
				playerState.Scores.Selected = 0
			}
			stateMu.Unlock()

			hasExtraDice = extra
			broadcast(clients, data.CraftUpdateScore(playerState.Info.UserID, playerState.Scores))

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

			broadcast(clients, data.CraftScoreRoll(playerState.Info.UserID))
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

			broadcast(clients, data.CraftEndTurn(playerState.Info.UserID))
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
