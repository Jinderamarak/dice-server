package data

import (
	"dice-server/game/common/client"
	"dice-server/game/farkle/connect"
	"github.com/google/uuid"
)

const (
	VarPleaseReady   = "farkle-please-ready"
	VarPlayerReady   = "farkle-player-ready"
	VarPlayerJoining = "farkle-player-joining"
	VarGameBegin     = "farkle-game-begin"
	VarPleaseSync    = "farkle-please-sync"
	VarSyncState     = "farkle-sync-state"
	VarTurnBegin     = "farkle-turn-begin"
	VarDiceRoll      = "farkle-dice-roll"
	VarDiceTouch     = "farkle-dice-touch"
	VarDiceTouched   = "farkle-dice-touched"
	VarUpdateScore   = "farkle-update-score"
	VarTurnTimeout   = "farkle-turn-timeout"
	VarScoreRoll     = "farkle-score-roll"
	VarEndTurn       = "farkle-end-turn"
	VarGameEnd       = "farkle-game-end"
	VarError         = "farkle-error"
	VarTerminate     = "farkle-terminate"
)

type VariantPleaseReady struct {
}

type VariantPlayerReady struct {
	PlayerID uuid.UUID `json:"playerId"`
}

type VariantPlayerJoining struct {
	Info *connect.LobbyPlayer `json:"info"`
}

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
	Dice     []*Dice `json:"dice"`
	Busted   bool    `json:"busted"`
	Deadline int64   `json:"deadline"`
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

type VariantError struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type VariantTerminate struct {
	Reason string `json:"reason"`
}

func CraftPleaseReady() *client.Message {
	return client.MustCraftMessage(VarPleaseReady, VariantPleaseReady{})
}

func CraftPlayerReady(playerID uuid.UUID) *client.Message {
	return client.MustCraftMessage(VarPlayerReady, VariantPlayerReady{PlayerID: playerID})
}

func CraftPlayerJoining(info *connect.LobbyPlayer) *client.Message {
	return client.MustCraftMessage(VarPlayerJoining, VariantPlayerJoining{Info: info})
}

func CraftGameBegin(state *GameState) *client.Message {
	return client.MustCraftMessage(VarGameBegin, VariantGameBegin{State: state})
}

func CraftPleaseSync() *client.Message {
	return client.MustCraftMessage(VarPleaseSync, VariantPleaseSync{})
}

func CraftSyncState(state *GameState) *client.Message {
	return client.MustCraftMessage(VarSyncState, VariantSyncState{State: state})
}

func CraftTurnBegin(playerID uuid.UUID) *client.Message {
	return client.MustCraftMessage(VarTurnBegin, VariantTurnBegin{PlayerID: playerID})
}

func CraftDiceRoll(dice []*Dice, busted bool, deadline int64) *client.Message {
	return client.MustCraftMessage(VarDiceRoll, VariantDiceRoll{Dice: dice, Busted: busted, Deadline: deadline})
}

func CraftDiceTouch(diceID uuid.UUID, selected bool) *client.Message {
	return client.MustCraftMessage(VarDiceTouch, VariantDiceTouch{DiceID: diceID, Selected: selected})
}

func CraftDiceTouched(playerID uuid.UUID, dice []*Dice) *client.Message {
	return client.MustCraftMessage(VarDiceTouched, VariantDiceTouched{PlayerID: playerID, Dice: dice})
}

func CraftUpdateScore(playerID uuid.UUID, scores PlayerScores) *client.Message {
	return client.MustCraftMessage(VarUpdateScore, VariantUpdateScore{PlayerID: playerID, Scores: scores})
}

func CraftTurnTimeout(playerID uuid.UUID) *client.Message {
	return client.MustCraftMessage(VarTurnTimeout, VariantTurnTimeout{PlayerID: playerID})
}

func CraftScoreRoll(playerID uuid.UUID) *client.Message {
	return client.MustCraftMessage(VarScoreRoll, VariantScoreRoll{PlayerID: playerID})
}

func CraftEndTurn(playerID uuid.UUID) *client.Message {
	return client.MustCraftMessage(VarEndTurn, VariantEndTurn{PlayerID: playerID})
}

func CraftGameEnd(winnerID uuid.UUID) *client.Message {
	return client.MustCraftMessage(VarGameEnd, VariantGameEnd{WinnerID: winnerID})
}

func CraftError(kind, message string) *client.Message {
	return client.MustCraftMessage(VarError, VariantError{Kind: kind, Message: message})
}

func CraftTerminate(reason string) *client.Message {
	return client.MustCraftMessage(VarTerminate, VariantTerminate{Reason: reason})
}
