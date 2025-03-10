package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"strconv"
	"strings"
)

type GameClient struct {
	conn     *websocket.Conn
	incoming chan string
}

func (client *GameClient) SendMessage(message string) {
	client.conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (client *GameClient) Loop() {
	defer client.conn.Close()
	for {
		variant, message, err := client.conn.ReadMessage()
		if err != nil {
			break
		}

		if variant != websocket.TextMessage {
			continue
		}

		client.incoming <- string(message)
	}
}

type GameSession struct {
	players [2]*GameClient
}

func (session *GameSession) broadcast(message string) {
	for _, player := range session.players {
		player.SendMessage(message)
	}
}

func (session *GameSession) broadcastExcept(message string, except int) {
	for i, player := range session.players {
		if i != except {
			player.SendMessage(message)
		}
	}
}

func (session *GameSession) Start() {
	session.broadcast("start")
	session.waitForReady(false, false)
}

func (session *GameSession) waitForReady(p1, p2 bool) {
	if p1 && p2 {
		session.startTurnRandom()
		return
	}

	select {
	case message := <-session.players[0].incoming:
		if message == "ready" {
			session.waitForReady(true, p2)
		}
	case message := <-session.players[1].incoming:
		if message == "ready" {
			session.waitForReady(p1, true)
		}
	}
}

func (session *GameSession) startTurnRandom() {
	randPlayer := rand.Intn(len(session.players))
	session.startTurn(randPlayer)
}

func (session *GameSession) startTurn(player int) {
	session.players[player].SendMessage("turn you")
	session.broadcastExcept("turn opponent", player)

	session.broadcast("dice 1,2,3,4,5,6")
	picks := session.getPicks(player)

	score := 0
	for _, pick := range picks {
		score += (pick + 1) * 10
	}
	session.broadcast(fmt.Sprintf("score %d", score))

	if score >= 100 {
		session.finishGame(player)
		return
	}

	session.startTurn((player + 1) % len(session.players))
}

func (session *GameSession) getPicks(player int) []int {
	picks := make([]int, 0)
	for {
		msg := <-session.players[player].incoming
		session.broadcastExcept(msg, player)
		if strings.HasPrefix(msg, "pick ") {
			pick, _ := strconv.Atoi(strings.TrimPrefix(msg, "pick "))
			picks = append(picks, pick)
		} else if msg == "done" {
			break
		}
	}

	return picks
}

func (session *GameSession) finishGame(winner int) {
	session.players[winner].SendMessage("winner you")
	session.broadcastExcept("winner opponent", winner)
}
