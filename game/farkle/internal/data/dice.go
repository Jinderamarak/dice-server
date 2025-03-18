package data

import (
	"github.com/google/uuid"
	"math/rand/v2"
)

type diceRoller interface {
	IntN(n int) int
}

var Roller diceRoller = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

type Dice struct {
	Id       uuid.UUID `json:"id"`
	Value    int       `json:"value"`
	Selected bool      `json:"selected"`
	Playable bool      `json:"playable"`
}

func NewDice(id uuid.UUID) *Dice {
	return &Dice{
		Id:       id,
		Value:    1,
		Selected: false,
		Playable: true,
	}
}

func (dice *Dice) Roll() Dice {
	if dice.Playable {
		return Dice{
			Id:       dice.Id,
			Value:    Roller.IntN(6) + 1,
			Selected: dice.Selected,
			Playable: dice.Playable,
		}
	} else {
		return Dice{
			Id:       dice.Id,
			Value:    dice.Value,
			Selected: dice.Selected,
			Playable: dice.Playable,
		}
	}
}
