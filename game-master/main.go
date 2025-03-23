package main

import (
	"dice-server/common/auth/token"
	"dice-server/common/utility"
	"dice-server/game/farkle/connect"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"time"
)

var rabbitConn *amqp.Connection
var rabbitChannel *amqp.Channel
var farkleCreateQueue amqp.Queue

func main() {
	var err error

	if rabbitConn, err = amqp.Dial("amqp://guest:guest@localhost:5672/"); err != nil {
		panic(err)
	}
	defer utility.CloseAndIgnore(rabbitConn)

	if rabbitChannel, err = rabbitConn.Channel(); err != nil {
		panic(err)
	}
	defer utility.CloseAndIgnore(rabbitChannel)

	if farkleCreateQueue, err = rabbitChannel.QueueDeclare(
		connect.CreateLobbyQueue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		panic(err)
	}

	server := gin.Default()
	server.Use(corsMiddleware())
	server.POST("/api/farkle", createFarkleHandler)
	server.POST("/api/farkle/:gameId/join", joinFarkleHandler)
	if err := server.Run("0.0.0.0:9000"); err != nil {
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

func publishCreateGameFarkle(data connect.CreateLobbyMessage) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err = rabbitChannel.Publish(
		"",
		farkleCreateQueue.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         dataBytes,
		}); err != nil {
		return err
	}

	return nil
}

func publishJoinGameFarkle(gameID uuid.UUID, data connect.JoinLobbyMessage) error {
	queue, err := rabbitChannel.QueueDeclare(
		connect.JoinLobbyQueue(gameID),
		false,
		true,
		false,
		false,
		amqp.Table{
			"x-expires": int32(1000 * 10),
		},
	)
	if err != nil {
		return err
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	log.Println("Publishing join game message to:", queue.Name)
	if err = rabbitChannel.Publish(
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        dataBytes,
		}); err != nil {
		return err
	}

	return nil
}

func createFarkleHandler(ctx *gin.Context) {
	gameID := uuid.New()
	playerID := uuid.New()

	createLobby := connect.CreateLobbyMessage{
		GameID: gameID,
		Target: 3000,
		Player: connect.LobbyPlayer{
			UserID:   playerID,
			Username: "Player 1",
			DiceSet: []connect.LobbyDice{
				{uuid.New()},
				{uuid.New()},
				{uuid.New()},
				{uuid.New()},
				{uuid.New()},
				{uuid.New()},
			},
		},
	}

	if err := publishCreateGameFarkle(createLobby); err != nil {
		log.Println("Failed to create game:", err)
		ctx.JSON(500, gin.H{"error": "Failed to create game"})
		return
	}

	gameToken := token.NewGameToken(playerID, gameID, "gamemaster", time.Now(), time.Now().Add(time.Hour))
	tokenString, err := gameToken.Sign([]byte(token.SuperSecret))
	if err != nil {
		log.Println("Failed to sign token:", err)
		ctx.JSON(500, gin.H{"error": "Failed to sign token"})
		return
	}

	ctx.JSON(201, gin.H{"token": tokenString})
}

func joinFarkleHandler(ctx *gin.Context) {
	gameIDStr := ctx.Param("gameId")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "Invalid game ID"})
		return
	}

	playerID := uuid.New()
	joinLobby := connect.JoinLobbyMessage{
		Player: connect.LobbyPlayer{
			UserID:   playerID,
			Username: "Player 2",
			DiceSet: []connect.LobbyDice{
				{uuid.New()},
				{uuid.New()},
				{uuid.New()},
				{uuid.New()},
				{uuid.New()},
				{uuid.New()},
			},
		},
	}

	if err := publishJoinGameFarkle(gameID, joinLobby); err != nil {
		log.Println("Failed to join game:", err)
		ctx.JSON(500, gin.H{"error": "Failed to join game"})
		return
	}

	gameToken := token.NewGameToken(playerID, gameID, "gamemaster", time.Now(), time.Now().Add(time.Hour))
	tokenString, err := gameToken.Sign([]byte(token.SuperSecret))
	if err != nil {
		log.Println("Failed to sign token:", err)
		ctx.JSON(500, gin.H{"error": "Failed to sign token"})
		return
	}

	ctx.JSON(200, gin.H{"token": tokenString})
}
