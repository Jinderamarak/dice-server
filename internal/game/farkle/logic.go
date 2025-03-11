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
	Roller       DiceRoller
}

func NewGame(p1, p2 *Player, roller DiceRoller) *Game {
	return &Game{
		Players:      [2]*Player{p1, p2},
		WinningScore: WinningScore,
		Roller:       roller,
	}
}

func (game *Game) broadcast(variant string, data interface{}) error {
	msg, err := client.MarshalMessage(variant, data)
	if err != nil {
		return err
	}

	for _, player := range game.Players {
		err := player.Client.SendMessage(msg)
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

		data := common.VariantError{
			Message: err.Error(),
		}

		msg, err := client.MarshalMessage(common.VarError, data)
		if err != nil {
			log.Println("Failed to marshal error message")
			return
		}

		for _, player := range game.Players {
			err := player.Client.SendMessage(msg)
			if err != nil {
				log.Println("Failed to send error message")
			}
		}
	}
}

func (game *Game) begin() error {
	err := game.broadcast(common.VarGameBegin, common.VariantGameBegin{
		Players: []uuid.UUID{game.Players[0].Id, game.Players[1].Id},
	})
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

		if err := game.broadcast(VarTurnBegin, VariantTurnBegin{
			PlayerId: player.Id,
		}); err != nil {
			return err
		}

		score, err := game.turnLoop(player)
		if err != nil {
			return err
		}

		player.Score += score
		if err := game.broadcast(VarUpdateScore, VariantUpdateScore{
			PlayerId:      player.Id,
			SelectedScore: 0,
			TurnScore:     score,
			TotalScore:    player.Score,
		}); err != nil {
			return err
		}

		if player.Score >= game.WinningScore {
			log.Println("Player won the game")
			if err := game.broadcast(common.VarGameEnd, common.VariantGameEnd{
				Winners: []uuid.UUID{player.Id},
			}); err != nil {
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
	availableDice := player.GetDice(player.DiceIds())

	for {
		log.Println("Rolling dice")
		for i, die := range availableDice {
			availableDice[i] = game.Roller.RollDie(die)
		}

		busted := hasBusted(countValues(availableDice))
		if err := game.broadcast(VarDiceRoll, VariantDiceRoll{
			Dice:   availableDice,
			Busted: busted,
		}); err != nil {
			return 0, err
		}

		if busted {
			log.Println("Player busted")
			return 0, nil
		}

		selectedScore := 0
		selectedExtra := false
		selectedDice := make([]Die, 0)
		rollAgain := false
		for {
			log.Println("Waiting for players next step")
			nextStep, err := player.Client.ReadMessage()
			if err != nil {
				return 0, err
			}
			log.Println("Next step variant:", nextStep.Variant)

			exitSelection := false
			switch nextStep.Variant {
			case VarDiceTouch:
				var data VariantDiceTouch
				err := nextStep.UnmarshalData(&data)
				if err != nil {
					return 0, err
				}

				log.Println("Player touched die:", data.DieId, data.Selected)
				err = touchDie(&availableDice, &selectedDice, data.DieId, data.Selected)
				if err != nil {
					message, err := client.MarshalMessage(common.VarError, common.VariantError{
						Message: err.Error(),
					})
					if err != nil {
						return 0, err
					}

					err = player.Client.SendMessage(message)
					if err != nil {
						return 0, err
					}
				}

				if err := game.broadcast(VarDiceTouch, &data); err != nil {
					return 0, err
				}

				selectedScore, selectedExtra = scoreCounts(countValues(selectedDice))
				realSelectedScore := selectedScore
				if selectedExtra {
					selectedScore = 0
				}

				log.Println("Updated scores:", selectedScore, realSelectedScore, turnScore, player.Score)
				if err := game.broadcast(VarUpdateScore, VariantUpdateScore{
					PlayerId:      player.Id,
					SelectedScore: realSelectedScore,
					TurnScore:     turnScore,
					TotalScore:    player.Score,
				}); err != nil {
					return 0, err
				}
			case VarScoreRoll:
				if len(selectedDice) == 0 {
					message, err := client.MarshalMessage(common.VarError, common.VariantError{
						Message: "No dice selected",
					})
					if err != nil {
						return 0, err
					}

					err = player.Client.SendMessage(message)
					if err != nil {
						return 0, err
					}
					continue
				}

				if selectedExtra {
					message, err := client.MarshalMessage(common.VarError, common.VariantError{
						Message: "Selected extra dice",
					})
					if err != nil {
						return 0, err
					}

					err = player.Client.SendMessage(message)
					if err != nil {
						return 0, err
					}
					continue
				}

				rollAgain = true
				exitSelection = true
			case VarEndTurn:
				if selectedExtra {
					message, err := client.MarshalMessage(common.VarError, common.VariantError{
						Message: "Selected extra dice",
					})
					if err != nil {
						return 0, err
					}

					err = player.Client.SendMessage(message)
					if err != nil {
						return 0, err
					}
					continue
				}

				rollAgain = false
				exitSelection = true
			}

			if exitSelection {
				break
			}
		}

		turnScore += selectedScore
		if !rollAgain {
			break
		}

		if len(availableDice) == 0 {
			availableDice = player.GetDice(player.DiceIds())
		}
	}

	return turnScore, nil
}

func touchDie(available *[]Die, selected *[]Die, dieId uuid.UUID, sel bool) error {
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
