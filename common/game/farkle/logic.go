package farkle

import (
	"dice-server/common/client"
	"dice-server/common/game/common"
	"errors"
	"github.com/google/uuid"
	"log"
	"math/rand"
	"time"
)

const EndSleep = 10 * time.Second

func NewGame(p1, p2 *Player, winningScore int) *Game {
	return &Game{
		Players:      [2]*Player{p1, p2},
		WinningScore: winningScore,
	}
}

func (game *Game) broadcast(message *client.Message) {
	for _, player := range game.Players {
		player.Client.SendMessage(message)
	}
}

func (game *Game) Start() {
	defer func() {
		for _, p := range game.Players {
			p.Client.Close()
		}
	}()

	log.Println("Starting game of Farkle with score ", game.WinningScore)
	err := game.begin()
	if err != nil {
		log.Println("Game terminated:", err)

		errorMessage := common.MakeError(err.Error())
		for _, player := range game.Players {
			player.Client.SendMessage(errorMessage)
		}
	}

	log.Println("Game ended, waiting")
	time.Sleep(EndSleep)
	log.Println("Game closed")
}

func (game *Game) begin() error {
	game.broadcast(common.MakeGameBegin(
		game.Players[0].Id,
		game.Players[1].Id,
	))

	return game.gameLoop()
}

func (game *Game) gameLoop() error {
	currentPlayer := rand.Intn(len(game.Players))
	for {
		currentPlayer = (currentPlayer + 1) % len(game.Players)
		player := game.Players[currentPlayer]

		game.broadcast(MakeTurnBegin(player.Id))

		score, err := game.turnLoop(player)
		if err != nil {
			return err
		}

		player.Score += score
		game.broadcast(MakeUpdateScore(
			player.Id,
			0,
			score,
			player.Score,
		))

		if player.Score >= game.WinningScore {
			log.Println("Player won the game")
			game.broadcast(common.MakeGameEnd(player.Id))
			break
		}
	}

	return nil
}

func (game *Game) turnLoop(player *Player) (int, error) {
	log.Println("New turn for player", player.Id)

	turnScore := 0
	availableDice := player.CopyOfDice()

	for {
		log.Println("Rolling dice")
		for i, dice := range availableDice {
			availableDice[i] = dice.Roll()
		}

		busted := hasBusted(countValues(availableDice))
		game.broadcast(MakeDiceRoll(availableDice, busted))

		if busted {
			log.Println("Player busted")
			return 0, nil
		}

		selectedScore := 0
		selectedExtra := false
		selectedDice := make([]common.Dice, 0)
		rollAgain := false
		for {
			log.Println("Waiting for players next step")
			nextStep, err := player.Client.ReadMessage(time.Second * 60)
			if err != nil {
				if errors.Is(err, client.ErrReadTimeout) {
					return 0, nil
				}
				return 0, err
			}
			log.Println("Next step variant:", nextStep.Variant)

			endSelection := false
			switch nextStep.Variant {
			case VarDiceTouch:
				var data VariantDiceTouch
				if err := nextStep.UnmarshalData(&data); err != nil {
					return 0, err
				}

				log.Println("Player touched dice:", data.DiceId, data.Selected)
				if !touchDice(&availableDice, &selectedDice, data.DiceId, data.Selected) {
					player.Client.SendMessage(common.MakeError("Unknown dice"))
				}

				game.broadcast(MakeDiceTouch(data.DiceId, data.Selected))

				selectedScore, selectedExtra = scoreCounts(countValues(selectedDice))
				realSelectedScore := selectedScore
				if selectedExtra {
					selectedScore = 0
				}

				log.Println("Updated scores:", selectedScore, realSelectedScore, turnScore, player.Score)
				game.broadcast(MakeUpdateScore(
					player.Id,
					realSelectedScore,
					turnScore,
					player.Score,
				))
			case VarScoreRoll:
				if len(selectedDice) == 0 {
					player.Client.SendMessage(common.MakeError("No dice selected"))
					continue
				}

				if selectedExtra {
					player.Client.SendMessage(common.MakeError("Selected extra dice"))
					continue
				}

				game.broadcast(MakeScoreRoll(player.Id))

				rollAgain = true
				endSelection = true
			case VarEndTurn:
				if selectedExtra {
					player.Client.SendMessage(common.MakeError("Selected extra dice"))
					continue
				}

				game.broadcast(MakeEndTurn(player.Id))

				rollAgain = false
				endSelection = true
			}

			if endSelection {
				break
			}
		}

		turnScore += selectedScore
		if !rollAgain {
			break
		}

		if len(availableDice) == 0 {
			availableDice = player.CopyOfDice()
		}
	}

	return turnScore, nil
}

func touchDice(available *[]common.Dice, selected *[]common.Dice, diceId uuid.UUID, sel bool) bool {
	if sel {
		for i, dice := range *available {
			if dice.Id == diceId {
				*selected = append(*selected, dice)
				*available = append((*available)[:i], (*available)[i+1:]...)
				return true
			}
		}
	} else {
		for i, dice := range *selected {
			if dice.Id == diceId {
				*available = append(*available, dice)
				*selected = append((*selected)[:i], (*selected)[i+1:]...)
				return true
			}
		}
	}
	return false
}
