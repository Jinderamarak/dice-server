package farkle

import (
	"dice-server/internal/client"
	"dice-server/internal/game/common"
	"errors"
	"github.com/google/uuid"
	"log"
	"math/rand"
)

type Game struct {
	Players      [2]*Player
	WinningScore int
}

func NewGame(p1, p2 *Player) *Game {
	return &Game{
		Players:      [2]*Player{p1, p2},
		WinningScore: WinningScore,
	}
}

func (game *Game) broadcast(message *client.Message) error {
	for _, player := range game.Players {
		err := player.Client.SendMessage(message)
		if err != nil {
			return err
		}
	}
	return nil
}

func (game *Game) Start() {
	log.Println("Starting game of Farkle")
	err := game.begin()
	if err != nil {
		log.Println(err)

		errorMessage := common.MakeError(err.Error())
		for _, player := range game.Players {
			err := player.Client.SendMessage(errorMessage)
			if err != nil {
				log.Println("Failed to send error message")
			}
		}
	}
}

func (game *Game) begin() error {
	err := game.broadcast(common.MakeGameBegin(
		game.Players[0].Id,
		game.Players[1].Id,
	))
	if err != nil {
		return err
	}

	return game.gameLoop()
}

func (game *Game) gameLoop() error {
	currentPlayer := rand.Intn(len(game.Players))
	for {
		currentPlayer = (currentPlayer + 1) % len(game.Players)
		player := game.Players[currentPlayer]

		if err := game.broadcast(MakeTurnBegin(player.Id)); err != nil {
			return err
		}

		score, err := game.turnLoop(player)
		if err != nil {
			return err
		}

		player.Score += score
		if err := game.broadcast(MakeUpdateScore(
			player.Id,
			0,
			score,
			player.Score,
		)); err != nil {
			return err
		}

		if player.Score >= game.WinningScore {
			log.Println("Player won the game")
			if err := game.broadcast(common.MakeGameEnd(player.Id)); err != nil {
				return err
			}
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
		if err := game.broadcast(MakeDiceRoll(availableDice, busted)); err != nil {
			return 0, err
		}

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
			nextStep, err := player.Client.ReadMessage()
			if err != nil {
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
				if err := touchDie(&availableDice, &selectedDice, data.DiceId, data.Selected); err != nil {
					err = player.Client.SendMessage(common.MakeError(err.Error()))
					if err != nil {
						return 0, err
					}
				}

				if err := game.broadcast(MakeDiceTouch(data.DiceId, data.Selected)); err != nil {
					return 0, err
				}

				selectedScore, selectedExtra = scoreCounts(countValues(selectedDice))
				realSelectedScore := selectedScore
				if selectedExtra {
					selectedScore = 0
				}

				log.Println("Updated scores:", selectedScore, realSelectedScore, turnScore, player.Score)
				if err := game.broadcast(MakeUpdateScore(
					player.Id,
					realSelectedScore,
					turnScore,
					player.Score,
				)); err != nil {
					return 0, err
				}
			case VarScoreRoll:
				if len(selectedDice) == 0 {
					if err := player.Client.SendMessage(common.MakeError("No dice selected")); err != nil {
						return 0, err
					}
					continue
				}

				if selectedExtra {
					if err := player.Client.SendMessage(common.MakeError("Selected extra dice")); err != nil {
						return 0, err
					}
					continue
				}

				if err := game.broadcast(MakeScoreRoll(player.Id)); err != nil {
					return 0, err
				}

				rollAgain = true
				endSelection = true
			case VarEndTurn:
				if selectedExtra {
					if err := player.Client.SendMessage(common.MakeError("Selected extra dice")); err != nil {
						return 0, err
					}
					continue
				}

				if err := game.broadcast(MakeEndTurn(player.Id)); err != nil {
					return 0, err
				}

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

func touchDie(available *[]common.Dice, selected *[]common.Dice, dieId uuid.UUID, sel bool) error {
	if sel {
		for i, die := range *available {
			if die.Id == dieId {
				*selected = append(*selected, die)
				*available = append((*available)[:i], (*available)[i+1:]...)
				return nil
			}
		}
		return errors.New("die not found")
	} else {
		for i, die := range *selected {
			if die.Id == dieId {
				*available = append(*available, die)
				*selected = append((*selected)[:i], (*selected)[i+1:]...)
				return nil
			}
		}
		return errors.New("die not found")
	}
}
