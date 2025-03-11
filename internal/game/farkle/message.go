package farkle

import (
	"dice-server/internal/client"
	"github.com/google/uuid"
)

const (
	VarTurnBegin   = "farkle-turn-begin"
	VarDiceRoll    = "farkle-dice-roll"
	VarUpdateScore = "farkle-update-score"
)

const (
	VarDiceTouch = "farkle-dice-touch"
	VarScoreRoll = "farkle-score-roll"
	VarEndTurn   = "farkle-end-turn"
)

type VariantTurnBegin struct {
	PlayerId uuid.UUID `json:"playerId"`
}

type VariantDiceRoll struct {
	Dice   []Die `json:"dice"`
	Busted bool  `json:"busted"`
}

type VariantUpdateScore struct {
	PlayerId      uuid.UUID `json:"playerId"`
	SelectedScore int       `json:"selectedScore"`
	TurnScore     int       `json:"turnScore"`
	TotalScore    int       `json:"totalScore"`
}

type VariantDiceTouch struct {
	DieId    uuid.UUID `json:"dieId"`
	Selected bool      `json:"selected"`
}

type VariantScoreRoll struct {
	PlayerId uuid.UUID `json:"playerId"`
}

type VariantEndTurn struct {
	PlayerId uuid.UUID `json:"playerId"`
}

func MakeTurnBegin(playerId uuid.UUID) *client.Message {
	return client.MustMarshalMessage(VarTurnBegin, VariantTurnBegin{PlayerId: playerId})
}

func MakeDiceRoll(dice []Die, busted bool) *client.Message {
	return client.MustMarshalMessage(VarDiceRoll, VariantDiceRoll{Dice: dice, Busted: busted})
}

func MakeUpdateScore(playerId uuid.UUID, selectedScore, turnScore, totalScore int) *client.Message {
	return client.MustMarshalMessage(VarUpdateScore, VariantUpdateScore{
		PlayerId:      playerId,
		SelectedScore: selectedScore,
		TurnScore:     turnScore,
		TotalScore:    totalScore,
	})
}

func MakeDiceTouch(dieId uuid.UUID, selected bool) *client.Message {
	return client.MustMarshalMessage(VarDiceTouch, VariantDiceTouch{DieId: dieId, Selected: selected})
}

func MakeScoreRoll(playerId uuid.UUID) *client.Message {
	return client.MustMarshalMessage(VarScoreRoll, VariantScoreRoll{PlayerId: playerId})
}

func MakeEndTurn(playerId uuid.UUID) *client.Message {
	return client.MustMarshalMessage(VarEndTurn, VariantEndTurn{PlayerId: playerId})
}
