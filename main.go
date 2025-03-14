package main

import (
	"dice-server/internal/client"
	"dice-server/internal/game/common"
	"dice-server/internal/game/farkle"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"sync"
)

var manager = client.NewClientsManager(websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
})

func main() {
	go manager.ClosureHandler()

	server := gin.Default()
	server.Use(CORSMiddleware())
	server.GET("/api/game/:id", gameHandler)
	errors.Unwrap(server.Run("0.0.0.0:9000"))
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

type tempPlayer struct {
	id          uuid.UUID
	c           *client.WebSocketClient
	targetScore int
}

var clientsMu = sync.Mutex{}
var clients = make(map[string]tempPlayer)

func gameHandler(ctx *gin.Context) {
	gameId := ctx.Param("id")

	playerIdStr := ctx.Query("playerId")
	playerId := uuid.MustParse(playerIdStr)

	targetScoreStr := ctx.Query("targetScore")
	targetScore, err := strconv.Atoi(targetScoreStr)
	if err != nil {
		targetScore = 3000
	}

	log.Println("Connection for game", gameId)
	currentClient, reconnected, _ := manager.Upgrade(playerId, ctx)
	if reconnected {
		log.Println("Reconnected")
		return
	}

	otherClient, ok := clients[gameId]
	if !ok || otherClient.c.IsClosed() {
		clientsMu.Lock()
		clients[gameId] = tempPlayer{
			id:          playerId,
			c:           currentClient,
			targetScore: targetScore,
		}
		clientsMu.Unlock()

		log.Println("Waiting for other player")
		return
	}

	clientsMu.Lock()
	delete(clients, gameId)
	clientsMu.Unlock()

	currentPlayer := farkle.NewPlayer(playerId, currentClient, common.NewRandomDiceSet(6))
	otherPlayer := farkle.NewPlayer(otherClient.id, otherClient.c, common.NewRandomDiceSet(6))

	game := farkle.NewGame(currentPlayer, otherPlayer, otherClient.targetScore)
	go game.Start()
}
