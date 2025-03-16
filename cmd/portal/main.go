package main

import (
	"dice-server/internal/auth/token"
	"dice-server/internal/channel"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"net/http"
)

const (
	serverHost = "0.0.0.0:9000"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var rabbitConnection *amqp.Connection

func main() {
	rabbit, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		errors.Unwrap(err)
	}

	rabbitConnection = rabbit
	defer func(rabbitConnection *amqp.Connection) {
		_ = rabbitConnection.Close()
	}(rabbitConnection)

	server := gin.Default()
	server.Use(CORSMiddleware())
	server.GET("/api/portal/:auth", portalHandler)
	errors.Unwrap(server.Run(serverHost))
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

func portalHandler(ctx *gin.Context) {
	auth := ctx.Param("auth")
	gameToken, err := token.ValidateGameToken(auth, []byte(token.SuperSecret))
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	wsChannel := channel.NewWebSocketChannel(conn)

	writeTopic := fmt.Sprintf("/game/%s/from/%s", gameToken.GameId, gameToken.UserId)
	readTopic := fmt.Sprintf("/game/%s/to/%s", gameToken.GameId, gameToken.UserId)
	rabbitChannel, err := channel.OpenRabbitChannel(rabbitConnection, writeTopic, readTopic)
	if err != nil {
		log.Println("failed to open rabbit channel:", err)
		wsChannel.Close()
		return
	}

	log.Printf("Opening portal between user %s and game %s\n", gameToken.UserId, gameToken.GameId)
	log.Printf("  - %s\n  - %s\n", writeTopic, readTopic)
	go openPortal(wsChannel, rabbitChannel)
}

func openPortal(web *channel.WebSocketChannel, rabbit *channel.RabbitChannel) {
	defer web.Close()
	defer rabbit.Close()

	for {
		select {
		case message := <-web.ReadChannel():
			err := rabbit.SendMessage(message)
			if err != nil {
				log.Println("failed to send message to rabbit:", err)
			}
		case message := <-rabbit.ReadChannel():
			err := web.WriteMessage(message)
			if err != nil {
				log.Println("failed to send message to websocket:", err)
			}
		case <-web.Closed():
			//	TODO: Send close to rabbit
			log.Println("closed by websocket")
			return
		case <-rabbit.Closed():
			//	TODO: Send close to websocket
			log.Println("closed by rabbit")
		}
	}
}
