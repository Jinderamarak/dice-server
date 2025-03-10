package main

import (
	"dice-server/game"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

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

var gameSessions = make(map[string]*game.GameSession)

func gameHandler(ctx *gin.Context) {
	gameId := ctx.Param("id")

	fmt.Println("Connection for game", gameId)
	conn, _ := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	client := &game.GameClient{Conn: conn, Incoming: make(chan string)}

	id := "idk"
	session, ok := gameSessions[gameId]
	if !ok {
		fmt.Println("Creating new session")
		session = &game.GameSession{[2]*game.GameClient{client, nil}}
		gameSessions[gameId] = session
		id = "p1"
	} else if session.Players[1] == nil {
		fmt.Println("Joining existing session and starting")
		session.Players[1] = client
		go session.Start()
		id = "p2"
	} else {
		client.Conn.Close()
		fmt.Println("Joining full game")
		return
	}

	client.Loop(id)
}
