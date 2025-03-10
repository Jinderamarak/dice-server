package main

import (
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

var gameSessions = make(map[string]*GameSession)

func gameHandler(ctx *gin.Context) {
	gameId := ctx.Param("id")

	conn, _ := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	client := &GameClient{conn: conn, incoming: make(chan string)}

	session, ok := gameSessions[gameId]
	if !ok {
		session = &GameSession{[2]*GameClient{client, nil}}
	} else if session.players[1] == nil {
		session.players[1] = client
		go session.Start()
	} else {
		client.conn.Close()
		fmt.Println("joining full game")
		return
	}

	client.Loop()
}
