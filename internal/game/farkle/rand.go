package farkle

import "math/rand"

type DiceRoller interface {
	RollDie(die Die) Die
}

type RandomDiceRoller struct{}

func (r *RandomDiceRoller) RollDie(die Die) Die {
	die.Value = rand.Intn(6) + 1
	return die
}
