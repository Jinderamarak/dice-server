package common

import (
	"github.com/google/uuid"
	"math/rand/v2"
)

type DiceRoller interface {
	IntN(n int) int
}

var Roller DiceRoller = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

type Dice struct {
	Id    uuid.UUID `json:"id"`
	Value int       `json:"value"`
}

func NewDice(id uuid.UUID) Dice {
	return Dice{
		Id:    id,
		Value: 6,
	}
}

func (dice *Dice) Roll() Dice {
	return Dice{
		Id:    dice.Id,
		Value: Roller.IntN(6) + 1,
	}
}

func NewRandomDiceSet(n int) []Dice {
	diceSet := make([]Dice, n)
	for i := 0; i < n; i++ {
		diceSet[i] = NewDice(uuid.New())
	}
	return diceSet
}
