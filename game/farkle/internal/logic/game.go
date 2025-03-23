package logic

import (
	"dice-server/common/channel"
	"dice-server/common/channel/message"
	"dice-server/game/farkle/internal/data"
	"errors"
	"log"
	"math/rand"
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
)

func PlayFarkle(state *data.GameState, clients []*data.PlayerClient) {
	defer func() {
		for _, client := range clients {
			client.Close()
		}
	}()

	for _, client := range clients {
		handler := func(msg *message.Message) {
			if msg.Variant == data.VarSyncState {
				_ = client.SendMessage(data.CraftSyncState(state))
			}
		}
		client.SetImportantHandler(&handler)
	}

	broadcast(clients, data.CraftGameBegin(state))
	time.Sleep(beginSleep)

	err := gameLoop(state, clients)
	if err != nil {
		log.Println("Game terminated:", err)
		broadcast(clients, message.CraftControlError(errGeneral, "server error"))
	}

	broadcast(clients, message.CraftControlTerminate("game closed"))
	time.Sleep(terminateSleep)
}

func broadcast(clients []*data.PlayerClient, msg *message.Message) {
	for _, client := range clients {
		err := client.SendMessage(msg)
		if err != nil {
			log.Println("Error broadcasting to client:", err)
		}
	}
}

func gameLoop(state *data.GameState, clients []*data.PlayerClient) error {
	currentPlayerIdx := rand.Intn(len(state.Players))
	for {
		currentPlayerIdx = (currentPlayerIdx + 1) % len(state.Players)
		currentPlayer := state.Players[currentPlayerIdx]
		currentClient := clients[currentPlayerIdx]

		state.CurrentPlayer = currentPlayer.Info.UserID

		err := turnLoop(clients, currentPlayer, currentClient)
		if err != nil {
			return err
		}

		broadcast(clients, data.CraftUpdateScore(currentPlayer.Info.UserID, currentPlayer.Scores))

		if currentPlayer.Scores.Total >= state.Target {
			log.Println("Game ended, winner:", currentPlayer.Info.Username)
			broadcast(clients, data.CraftGameEnd(currentPlayer.Info.UserID))
			return nil
		}
	}
}

func turnLoop(clients []*data.PlayerClient, player *data.PlayerState, client *data.PlayerClient) error {
	log.Println("New turn:", player.Info.Username)

	client.SetOnTurn()
	defer client.SetOffTurn()

	broadcast(clients, data.CraftTurnBegin(player.Info.UserID))
	time.Sleep(turnBeginSleep)

	player.Scores.Turn = 0
	player.Scores.Selected = 0
	resetDice(player.Dice)

	for {
		rollDice(player.Dice)
		busted := hasBusted(countValues(player.Dice, true))
		broadcast(clients, data.CraftDiceRoll(player.Dice, busted))

		if busted {
			log.Println("Player busted")
			player.Scores.Turn = 0
			player.Scores.Selected = 0
			broadcast(clients, data.CraftUpdateScore(player.Info.UserID, player.Scores))
			return nil
		}

		rollAgain, err := diceSelection(clients, player, client, time.Now().Add(pickTimeout))
		if err != nil {
			if errors.Is(err, channel.ErrReadTimeout) {
				player.Scores.Selected = 0
				player.Scores.Turn = 0
				broadcast(clients, data.CraftTurnTimeout(player.Info.UserID))
				return nil
			} else {
				return err
			}
		}

		player.Scores.Turn += player.Scores.Selected
		player.Scores.Selected = 0

		moveSelectedToUnplayable(player.Dice)

		if !rollAgain {
			player.Scores.Total += player.Scores.Turn
			return nil
		}

		broadcast(clients, data.CraftUpdateScore(player.Info.UserID, player.Scores))
	}
}

func diceSelection(clients []*data.PlayerClient, player *data.PlayerState, client *data.PlayerClient, deadline time.Time) (bool, error) {
	hasExtraDice := false
	for {
		log.Println("Waiting for next step")
		step, err := client.ReadMessage(deadline.Sub(time.Now()))
		if err != nil {
			return false, err
		}
		log.Println("Next step is:", step.Variant)

		switch step.Variant {
		case data.VarDiceTouch:
			var diceTouch data.VariantDiceTouch
			if err = step.UnmarshalData(&diceTouch); err != nil {
				log.Println("Error unmarshalling dice touch:", err)
				_ = client.SendMessage(message.CraftControlError(errBadData, "sent bad data"))
				continue
			}

			if !touchDice(player.Dice, &diceTouch) {
				log.Println("Player touched bad dice")
				_ = client.SendMessage(message.CraftControlError(errBadDice, "touched bad dice"))
				continue
			}

			broadcast(clients, data.CraftDiceTouched(player.Info.UserID, player.Dice))

			selected, extra := scoreCounts(countValues(player.Dice, false))
			player.Scores.Selected = selected
			if extra {
				player.Scores.Selected = 0
			}

			hasExtraDice = extra
			broadcast(clients, data.CraftUpdateScore(player.Info.UserID, player.Scores))

		case data.VarScoreRoll:
			var scoreRoll data.VariantScoreRoll
			if err = step.UnmarshalData(&scoreRoll); err != nil {
				log.Println("Error unmarshalling score roll:", err)
				_ = client.SendMessage(message.CraftControlError(errBadData, "sent bad data"))
				continue
			}

			if player.Info.UserID != scoreRoll.PlayerID {
				_ = client.SendMessage(message.CraftControlError(errBadPlayer, "bad player id"))
				continue
			}

			if player.Scores.Selected == 0 {
				_ = client.SendMessage(message.CraftControlError(errNoneSelected, "no dice selected"))
				continue
			}

			if hasExtraDice {
				_ = client.SendMessage(message.CraftControlError(errExtraDice, "extra dice selected"))
				continue
			}

			broadcast(clients, data.CraftScoreRoll(player.Info.UserID))
			return true, nil

		case data.VarEndTurn:
			var endTurn data.VariantEndTurn
			if err = step.UnmarshalData(&endTurn); err != nil {
				log.Println("Error unmarshalling end turn:", err)
				_ = client.SendMessage(message.CraftControlError(errBadData, "sent bad data"))
				continue
			}

			if player.Info.UserID != endTurn.PlayerID {
				_ = client.SendMessage(message.CraftControlError(errBadPlayer, "bad player id"))
				continue
			}

			if hasExtraDice {
				_ = client.SendMessage(message.CraftControlError(errExtraDice, "extra dice selected"))
				continue
			}

			broadcast(clients, data.CraftEndTurn(player.Info.UserID))
			return false, nil
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
