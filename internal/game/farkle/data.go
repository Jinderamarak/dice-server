package farkle

import (
	"dice-server/internal/game/common"
	"github.com/google/uuid"
)

type Player struct {
	Id      uuid.UUID
	DiceSet []common.Dice
	Score   int

	Client common.Client
}

func NewPlayer(id uuid.UUID, client common.Client, dice []common.Dice) *Player {
	if client == nil {
		return nil
	}

	return &Player{
		Id:      id,
		DiceSet: dice,
		Score:   0,
		Client:  client,
	}
}

func (p *Player) CopyOfDice() []common.Dice {
	diceCopy := make([]common.Dice, len(p.DiceSet))
	copy(diceCopy, p.DiceSet)
	return diceCopy
}
