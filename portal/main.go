package main

import (
	"dice-server/common/auth/token"
	"dice-server/common/channel"
	"dice-server/common/utility"
	"dice-server/portal/connect"
	wschan "dice-server/portal/internal/channel"
	"dice-server/portal/internal/loop"
	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"net/http"
)

const (
	serverHost = "0.0.0.0:9001"
)

var upgrader = ws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var rabbitConnection *amqp.Connection

func main() {
	rabbit, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		panic(err)
	}

	rabbitConnection = rabbit
	defer utility.CloseAndIgnore(rabbitConnection)

	server := gin.Default()
	server.Use(corsMiddleware())
	server.GET("/api/portal/:auth", portalHandler)
	if err := server.Run(serverHost); err != nil {
		panic(err)
	}
}

func corsMiddleware() gin.HandlerFunc {
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

	wsChannel := wschan.NewWebSocketChannel(conn)

	writeTopic := connect.PlayerToGameQueue(gameToken.GameId, gameToken.UserId)
	readTopic := connect.GameToPlayerQueue(gameToken.GameId, gameToken.UserId)
	rabbitChannel, err := channel.OpenRabbitChannel(rabbitConnection, writeTopic, readTopic)
	if err != nil {
		log.Println("Failed to open rabbit channel:", err)
		wsChannel.Close()
		return
	}

	log.Printf("Opening portal between user %s and game %s\n", gameToken.UserId, gameToken.GameId)
	go loop.OpenPortal(wsChannel, rabbitChannel, gameToken)
}
