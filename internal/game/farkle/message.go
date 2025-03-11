package farkle

import "github.com/google/uuid"

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
