package farkle

import (
	"dice-server/internal/client"
	"dice-server/internal/game/common"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"testing"
)

type TestClient struct {
	SendingQueue   []*client.Message
	ReceivingQueue []*client.Message
}

func (c *TestClient) SendMessage(message *client.Message) error {
	c.SendingQueue = append(c.SendingQueue, message)
	return nil
}

func (c *TestClient) ReadMessage() (*client.Message, error) {
	if len(c.ReceivingQueue) == 0 {
		return &client.Message{}, errors.New("no messages to read")
	}

	message := c.ReceivingQueue[0]
	c.ReceivingQueue = c.ReceivingQueue[1:]
	return message, nil
}

type TestDiceRoller struct {
	Values []int
}

func (r *TestDiceRoller) IntN(_ int) int {
	if len(r.Values) == 0 {
		return 0
	}

	value := r.Values[0]
	r.Values = r.Values[1:]
	return value - 1
}

func injectDiceRoller() {
	roller := &TestDiceRoller{Values: []int{1, 1, 1, 1, 1, 1}}
	common.Roller = roller
}

func preparePlayer() (*Player, *TestClient) {
	dice := make([]common.Dice, 6)
	for i := range dice {
		dice[i] = common.NewDice(uuid.New())
	}

	c := &TestClient{
		SendingQueue: []*client.Message{},
		ReceivingQueue: []*client.Message{
			MakeDiceTouch(dice[0].Id, true),
			MakeDiceTouch(dice[1].Id, true),
			MakeDiceTouch(dice[2].Id, true),
			MakeDiceTouch(dice[3].Id, true),
			MakeDiceTouch(dice[4].Id, true),
			MakeDiceTouch(dice[5].Id, true),
			MakeEndTurn(uuid.Nil),
		},
	}

	return NewPlayer(uuid.New(), c, dice), c
}

func TestGameFlow(t *testing.T) {
	p1, c1 := preparePlayer()
	p2, c2 := preparePlayer()
	injectDiceRoller()

	game := NewGame(p1, p2, 1000)
	game.Start()

	printFarkleMessages("Player 1", c1.SendingQueue)
	printFarkleMessages("Player 2", c2.SendingQueue)
}

func printFarkleMessages(title string, messages []*client.Message) {
	fmt.Println(title)
	for _, message := range messages {
		fmt.Print("  ")
		printFarkleMessage(message)
	}
}

func printFarkleMessage(message *client.Message) {
	switch message.Variant {
	case common.VarGameBegin:
		var data common.VariantGameBegin
		message.MustUnmarshalData(&data)
		fmt.Println("Game Begin", data)
	case common.VarGameEnd:
		var data common.VariantGameEnd
		message.MustUnmarshalData(&data)
		fmt.Println("Game End", data)
	case common.VarError:
		var data common.VariantError
		message.MustUnmarshalData(&data)
		fmt.Println("Error", data)
	case VarTurnBegin:
		var data VariantTurnBegin
		message.MustUnmarshalData(&data)
		fmt.Println("Farkle Turn Begin", data)
	case VarDiceRoll:
		var data VariantDiceRoll
		message.MustUnmarshalData(&data)
		fmt.Println("Farkle Dice Roll", data)
	case VarUpdateScore:
		var data VariantUpdateScore
		message.MustUnmarshalData(&data)
		fmt.Println("Farkle Update Score", data)
	case VarDiceTouch:
		var data VariantDiceTouch
		message.MustUnmarshalData(&data)
		fmt.Println("Farkle Dice Touch", data)
	case VarScoreRoll:
		var data VariantScoreRoll
		message.MustUnmarshalData(&data)
		fmt.Println("Farkle Score Roll", data)
	case VarEndTurn:
		var data VariantEndTurn
		message.MustUnmarshalData(&data)
		fmt.Println("Farkle End Turn", data)
	}
}
