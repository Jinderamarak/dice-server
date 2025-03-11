package main

import (
	"dice-server/internal/client"
	"dice-server/internal/game/common"
	"dice-server/internal/game/farkle"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var manager = client.NewClientsManager(websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
})

func main() {
	server := gin.Default()
	server.Use(CORSMiddleware())
	server.GET("/game/:id", gameHandler)
	errors.Unwrap(server.Run("localhost:8080"))
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

var clients = make(map[string]common.Client)

func gameHandler(ctx *gin.Context) {
	gameId := ctx.Param("id")

	fmt.Println("Connection for game", gameId)
	currentClient, reconnected, _ := manager.Upgrade(uuid.New(), ctx)
	if reconnected {
		fmt.Println("Reconnected")
		return
	}

	otherClient, ok := clients[gameId]
	if !ok {
		clients[gameId] = currentClient
		return
	}

	delete(clients, gameId)
	currentPlayer := farkle.RandomPlayer(currentClient)
	otherPlayer := farkle.RandomPlayer(otherClient)

	roller := farkle.RandomDiceRoller{}
	game := farkle.NewGame(currentPlayer, otherPlayer, &roller)
	game.Start()
}
