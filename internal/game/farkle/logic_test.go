package farkle

import (
	"dice-server/internal/client"
	"errors"
	"github.com/google/uuid"
	"testing"
)

type TestClient struct {
	SendingQueue   []client.Message
	ReceivingQueue []client.Message
}

func (c *TestClient) SendMessage(message client.Message) error {
	c.SendingQueue = append(c.SendingQueue, message)
	return nil
}

func (c *TestClient) ReadMessage() (client.Message, error) {
	if len(c.ReceivingQueue) == 0 {
		return client.Message{}, errors.New("no messages to read")
	}

	message := c.ReceivingQueue[0]
	c.ReceivingQueue = c.ReceivingQueue[1:]
	return message, nil
}

type TestDiceRoller struct {
	Values []int
}

func (r *TestDiceRoller) RollDie(die Die) Die {
	if len(r.Values) == 0 {
		die.Value = 1
		return die
	}

	die.Value = r.Values[0]
	r.Values = r.Values[1:]
	return die
}

func prepareDiceRoller() DiceRoller {
	return &TestDiceRoller{Values: []int{1, 1, 1, 1, 1, 1}}
}

func preparePlayer() *Player {
	dice := make([]Die, 6)
	for i, _ := range dice {
		dice[i] = Die{Id: uuid.New(), Value: 1}
	}

	c := &TestClient{
		SendingQueue: []client.Message{},
		ReceivingQueue: []client.Message{
			client.MustMarshalMessage(VarDiceTouch, VariantDiceTouch{
				DieId:    dice[0].Id,
				Selected: true,
			}),
			client.MustMarshalMessage(VarDiceTouch, VariantDiceTouch{
				DieId:    dice[1].Id,
				Selected: true,
			}),
			client.MustMarshalMessage(VarDiceTouch, VariantDiceTouch{
				DieId:    dice[2].Id,
				Selected: true,
			}),
			client.MustMarshalMessage(VarDiceTouch, VariantDiceTouch{
				DieId:    dice[3].Id,
				Selected: true,
			}),
			client.MustMarshalMessage(VarDiceTouch, VariantDiceTouch{
				DieId:    dice[4].Id,
				Selected: true,
			}),
			client.MustMarshalMessage(VarDiceTouch, VariantDiceTouch{
				DieId:    dice[5].Id,
				Selected: true,
			}),
			client.MustMarshalMessage(VarEndTurn, VariantEndTurn{}),
		},
	}

	return NewPlayer(c, dice)
}

func prepareGame() *Game {
	return NewGame(preparePlayer(), preparePlayer(), prepareDiceRoller())
}

func TestGameFlow(t *testing.T) {
	game := prepareGame()
	game.Start()
}
