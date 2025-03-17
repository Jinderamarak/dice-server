package data

import (
	"dice-server/common/channel/message"
	"github.com/google/uuid"
)

const (
	VarPlayerJoin  = "farkle-player-join"
	VarGameBegin   = "farkle-game-begin"
	VarSyncState   = "farkle-sync-state"
	VarTurnBegin   = "farkle-turn-begin"
	VarDiceRoll    = "farkle-dice-roll"
	VarDiceTouch   = "farkle-dice-touch"
	VarDiceTouched = "farkle-dice-touched"
	VarUpdateScore = "farkle-update-score"
	VarScoreRoll   = "farkle-score-roll"
	VarEndTurn     = "farkle-end-turn"
	VarGameEnd     = "farkle-game-end"
)

type VariantPlayerJoin struct {
	Player LobbyPlayer `json:"player"`
}

type VariantGameBegin struct {
	State GameState `json:"state"`
}

type VariantSyncState struct {
	State GameState `json:"state"`
}

type VariantTurnBegin struct {
	PlayerId uuid.UUID `json:"playerId"`
}

type VariantDiceRoll struct {
	Dice   []Dice `json:"dice"`
	Busted bool   `json:"busted"`
}

type VariantDiceTouch struct {
	DiceId   uuid.UUID `json:"diceId"`
	Selected bool      `json:"selected"`
}

type VariantDiceTouched struct {
	Dice []Dice `json:"dice"`
}

type VariantUpdateScore struct {
	PlayerId uuid.UUID    `json:"playerId"`
	Scores   PlayerScores `json:"scores"`
}

type VariantScoreRoll struct {
	PlayerId uuid.UUID `json:"playerId"`
}

type VariantEndTurn struct {
	PlayerId uuid.UUID `json:"playerId"`
}

type VariantGameEnd struct {
	WinnerId uuid.UUID `json:"winnerId"`
}

func CraftPlayerJoin(player LobbyPlayer) *message.Message {
	return message.MustCraftMessage(VarPlayerJoin, VariantPlayerJoin{Player: player})
}

func CraftGameBegin(state GameState) *message.Message {
	return message.MustCraftMessage(VarGameBegin, VariantGameBegin{State: state})
}

func CraftSyncState(state GameState) *message.Message {
	return message.MustCraftMessage(VarSyncState, VariantSyncState{State: state})
}

func CraftTurnBegin(playerId uuid.UUID) *message.Message {
	return message.MustCraftMessage(VarTurnBegin, VariantTurnBegin{PlayerId: playerId})
}

func CraftDiceRoll(dice []Dice, busted bool) *message.Message {
	return message.MustCraftMessage(VarDiceRoll, VariantDiceRoll{Dice: dice, Busted: busted})
}

func CraftDiceTouch(diceId uuid.UUID, selected bool) *message.Message {
	return message.MustCraftMessage(VarDiceTouch, VariantDiceTouch{DiceId: diceId, Selected: selected})
}

func CraftDiceTouched(dice []Dice) *message.Message {
	return message.MustCraftMessage(VarDiceTouched, VariantDiceTouched{Dice: dice})
}

func CraftUpdateScore(playerId uuid.UUID, scores PlayerScores) *message.Message {
	return message.MustCraftMessage(VarUpdateScore, VariantUpdateScore{PlayerId: playerId, Scores: scores})
}

func CraftScoreRoll(playerId uuid.UUID) *message.Message {
	return message.MustCraftMessage(VarScoreRoll, VariantScoreRoll{PlayerId: playerId})
}

func CraftEndTurn(playerId uuid.UUID) *message.Message {
	return message.MustCraftMessage(VarEndTurn, VariantEndTurn{PlayerId: playerId})
}

func CraftGameEnd(winnerId uuid.UUID) *message.Message {
	return message.MustCraftMessage(VarGameEnd, VariantGameEnd{WinnerId: winnerId})
}
