package data

import (
	"dice-server/common/channel/message"
	"github.com/google/uuid"
)

const (
	VarGameBegin   = "farkle-game-begin"
	VarPleaseSync  = "farkle-please-sync"
	VarSyncState   = "farkle-sync-state"
	VarTurnBegin   = "farkle-turn-begin"
	VarDiceRoll    = "farkle-dice-roll"
	VarDiceTouch   = "farkle-dice-touch"
	VarDiceTouched = "farkle-dice-touched"
	VarUpdateScore = "farkle-update-score"
	VarTurnTimeout = "farkle-turn-timeout"
	VarScoreRoll   = "farkle-score-roll"
	VarEndTurn     = "farkle-end-turn"
	VarGameEnd     = "farkle-game-end"
)

type VariantGameBegin struct {
	State *GameState `json:"state"`
}

type VariantPleaseSync struct {
}

type VariantSyncState struct {
	State *GameState `json:"state"`
}

type VariantTurnBegin struct {
	PlayerID uuid.UUID `json:"playerId"`
}

type VariantDiceRoll struct {
	Dice   []*Dice `json:"dice"`
	Busted bool    `json:"busted"`
}

type VariantDiceTouch struct {
	DiceID   uuid.UUID `json:"diceId"`
	Selected bool      `json:"selected"`
}

type VariantDiceTouched struct {
	PlayerID uuid.UUID `json:"playerId"`
	Dice     []*Dice   `json:"dice"`
}

type VariantUpdateScore struct {
	PlayerID uuid.UUID    `json:"playerId"`
	Scores   PlayerScores `json:"scores"`
}

type VariantTurnTimeout struct {
	PlayerID uuid.UUID `json:"playerId"`
}

type VariantScoreRoll struct {
	PlayerID uuid.UUID `json:"playerId"`
}

type VariantEndTurn struct {
	PlayerID uuid.UUID `json:"playerId"`
}

type VariantGameEnd struct {
	WinnerID uuid.UUID `json:"winnerId"`
}

func CraftGameBegin(state *GameState) *message.Message {
	return message.MustCraftMessage(VarGameBegin, VariantGameBegin{State: state})
}

func CraftPleaseSync() *message.Message {
	return message.MustCraftMessage(VarPleaseSync, VariantPleaseSync{})
}

func CraftSyncState(state *GameState) *message.Message {
	return message.MustCraftMessage(VarSyncState, VariantSyncState{State: state})
}

func CraftTurnBegin(playerID uuid.UUID) *message.Message {
	return message.MustCraftMessage(VarTurnBegin, VariantTurnBegin{PlayerID: playerID})
}

func CraftDiceRoll(dice []*Dice, busted bool) *message.Message {
	return message.MustCraftMessage(VarDiceRoll, VariantDiceRoll{Dice: dice, Busted: busted})
}

func CraftDiceTouch(diceID uuid.UUID, selected bool) *message.Message {
	return message.MustCraftMessage(VarDiceTouch, VariantDiceTouch{DiceID: diceID, Selected: selected})
}

func CraftDiceTouched(playerID uuid.UUID, dice []*Dice) *message.Message {
	return message.MustCraftMessage(VarDiceTouched, VariantDiceTouched{PlayerID: playerID, Dice: dice})
}

func CraftUpdateScore(playerID uuid.UUID, scores PlayerScores) *message.Message {
	return message.MustCraftMessage(VarUpdateScore, VariantUpdateScore{PlayerID: playerID, Scores: scores})
}

func CraftTurnTimeout(playerID uuid.UUID) *message.Message {
	return message.MustCraftMessage(VarTurnTimeout, VariantTurnTimeout{PlayerID: playerID})
}

func CraftScoreRoll(playerID uuid.UUID) *message.Message {
	return message.MustCraftMessage(VarScoreRoll, VariantScoreRoll{PlayerID: playerID})
}

func CraftEndTurn(playerID uuid.UUID) *message.Message {
	return message.MustCraftMessage(VarEndTurn, VariantEndTurn{PlayerID: playerID})
}

func CraftGameEnd(winnerID uuid.UUID) *message.Message {
	return message.MustCraftMessage(VarGameEnd, VariantGameEnd{WinnerID: winnerID})
}

func IsImportantMessage(msg *message.Message) bool {
	if message.IsControlMessage(msg) {
		return true
	}

	switch msg.Variant {
	case VarSyncState:
		return true
	default:
		return false
	}
}
