package game

import (
	"fmt"
	"github.com/gorilla/websocket"
	"math/rand"
	"strconv"
	"strings"
)

type GameClient struct {
	Conn     *websocket.Conn
	Incoming chan string
}

func (client *GameClient) SendMessage(message string) {
	client.Conn.WriteMessage(websocket.TextMessage, []byte(message))
}

func (client *GameClient) Loop(id string) {
	defer client.Conn.Close()
	for {
		variant, message, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		if variant != websocket.TextMessage {
			continue
		}

		str := strings.TrimSpace(string(message))

		fmt.Println(id, ">", str)
		client.Incoming <- str
	}
}

type GameSession struct {
	Players [2]*GameClient
}

func (session *GameSession) broadcast(message string) {
	for _, player := range session.Players {
		player.SendMessage(message)
	}
}

func (session *GameSession) broadcastExcept(message string, except int) {
	for i, player := range session.Players {
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
		fmt.Println("Players are ready")
		session.startTurnRandom()
		return
	}

	fmt.Println("Waiting for 'ready'", p1, p2)
	select {
	case message := <-session.Players[0].Incoming:
		if message == "ready" {
			session.waitForReady(true, p2)
		}
	case message := <-session.Players[1].Incoming:
		if message == "ready" {
			session.waitForReady(p1, true)
		}
	}
}

func (session *GameSession) startTurnRandom() {
	randPlayer := rand.Intn(len(session.Players))
	session.startTurn(randPlayer)
}

func (session *GameSession) startTurn(player int) {
	session.Players[player].SendMessage("turn you")
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

	session.startTurn((player + 1) % len(session.Players))
}

func (session *GameSession) getPicks(player int) []int {
	picks := make([]int, 0)
	for {
		msg := <-session.Players[player].Incoming
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
	session.Players[winner].SendMessage("winner you")
	session.broadcastExcept("winner opponent", winner)
}
