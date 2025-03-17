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
}

func NewDice(id uuid.UUID) Dice {
	return Dice{
		Id:       id,
		Value:    1,
		Selected: false,
	}
}

func (dice *Dice) Roll() Dice {
	return Dice{
		Id:       dice.Id,
		Value:    Roller.IntN(6) + 1,
		Selected: dice.Selected,
	}
}
