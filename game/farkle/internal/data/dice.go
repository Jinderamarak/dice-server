package data

import (
	"dice-server/game/farkle/connect"
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
	Variant  string    `json:"variant"`
}

func NewDice(id uuid.UUID, variant string) *Dice {
	return &Dice{
		ID:       id,
		Value:    1,
		Selected: false,
		Playable: true,
		Variant:  variant,
	}
}

func (dice *Dice) Roll() {
	if dice.Playable {
		dice.Value = rollWeightedDice(dice.Variant)
	}
}

func rollWeightedDice(variant string) int {
	weights, ok := connect.DiceDistributions[variant]
	if !ok {
		weights = connect.DiceDistributions[connect.DiceRegular]
	}

	totalWeight := 0
	for _, weight := range weights {
		totalWeight += weight
	}

	roll := Roller.IntN(totalWeight)

	currentWeight := 0
	for i, weight := range weights {
		currentWeight += weight
		if roll < currentWeight {
			return i + 1
		}
	}

	panic("rolled number out of bounds weight, check dice distributions for variant: " + variant)
}
