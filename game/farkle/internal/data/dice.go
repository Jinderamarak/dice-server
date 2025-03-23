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
	ID       uuid.UUID `json:"id"`
	Value    int       `json:"value"`
	Selected bool      `json:"selected"`
	Playable bool      `json:"playable"`
}

func NewDice(id uuid.UUID) *Dice {
	return &Dice{
		ID:       id,
		Value:    1,
		Selected: false,
		Playable: true,
	}
}

func (dice *Dice) Roll() {
	if dice.Playable {
		dice.Value = Roller.IntN(6) + 1
	}
}
