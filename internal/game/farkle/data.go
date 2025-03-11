package farkle

import (
	"dice-server/internal/game/common"
	"github.com/google/uuid"
	"slices"
)

const DicePerPlayer = 6
const WinningScore = 1000

type Die struct {
	Id    uuid.UUID `json:"id"`
	Value int       `json:"value"`
}

func RandomDie() Die {
	return Die{
		Id:    uuid.New(),
		Value: 1,
	}
}

type Player struct {
	Id      uuid.UUID
	DiceSet []Die
	Score   int

	Client common.Client
}

func NewPlayer(client common.Client, dice []Die) *Player {
	if client == nil {
		return nil
	}

	return &Player{
		Id:      uuid.New(),
		DiceSet: dice,
		Score:   0,
		Client:  client,
	}
}

func RandomPlayer(client common.Client) *Player {
	if client == nil {
		return nil
	}

	diceSet := make([]Die, DicePerPlayer)
	for i := 0; i < DicePerPlayer; i++ {
		diceSet[i] = RandomDie()
	}

	return NewPlayer(client, diceSet)
}

func (player *Player) DiceIds() []uuid.UUID {
	ids := make([]uuid.UUID, len(player.DiceSet))
	for i, die := range player.DiceSet {
		ids[i] = die.Id
	}

	return ids
}

func (player *Player) GetDice(ids []uuid.UUID) []Die {
	var dice []Die
	for _, die := range player.DiceSet {
		if slices.Contains(ids, die.Id) {
			dice = append(dice, die)
		}
	}

	return dice
}
