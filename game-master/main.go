package main

import (
	"dice-server/common/queue"
	"dice-server/common/utility"
	"dice-server/game-master/internal/config"
	"dice-server/game/farkle/connect"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"log"
	"net/http"
)

var queuePool *queue.Pool

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Panicln("Failed to load configuration:", err)
	}

	var err error
	if queuePool, err = queue.NewPool(
		config.Config.Rabbit.Connections,
		config.Config.Rabbit.Channels,
		config.Config.Rabbit.URL,
	); err != nil {
		log.Panicln("Queue pool creation failed:", err)
	}
	defer utility.CloseAndIgnore(queuePool)

	server := gin.Default()
	server.Use(corsMiddleware())
	server.POST("/api/game/farkle", createFarkleHandler)
	server.POST("/api/game/farkle/:gameId/join", joinFarkleHandler)

	host := fmt.Sprintf(":%d", config.Config.Port)
	if err = server.Run(host); err != nil {
		log.Panicln("Failed to start server:", err)
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

func createGameFarkle(data connect.CreateFarkleRequest) (*connect.CreateFarkleResponse, error) {
	publisher := queuePool.GetPublisher(connect.CreateLobbyQueue)
	defer publisher.Close()

	listener := queuePool.GetConsumer(connect.AcceptLobbyQueue(data.GameID))
	defer listener.Close()

	if err := publisher.PublishJSON(data); err != nil {
		return nil, errors.Wrap(err, "failed to publish create game message")
	}

	messages := listener.Consume()
	for msg := range messages {
		var accept connect.CreateFarkleResponse
		if err := json.Unmarshal(msg.Body, &accept); err != nil {
			log.Println("Failed to unmarshal accepted lobby message:", err)
			continue
		}

		if accept.GameID == data.GameID {
			log.Println("Game created successfully:", accept.GameID)
			return &accept, nil
		}
	}
	return nil, errors.New("failed to receive accepted lobby message")
}

func joinGameFarkle(gameID uuid.UUID, data connect.JoinFarkleRequest) (*connect.JoinFarkleResponse, error) {
	publisher := queuePool.GetPublisher(connect.JoinLobbyQueue(gameID))
	defer publisher.Close()

	listener := queuePool.GetConsumer(connect.JoinedLobbyQueue(gameID))
	defer listener.Close()

	if err := publisher.PublishJSON(data); err != nil {
		return nil, errors.Wrap(err, "failed to publish join game message")
	}

	messages := listener.Consume()
	for msg := range messages {
		var joined connect.JoinFarkleResponse
		if err := json.Unmarshal(msg.Body, &joined); err != nil {
			log.Println("Failed to unmarshal joined lobby message:", err)
			continue
		}

		return &joined, nil
	}
	return nil, errors.New("failed to receive joined lobby message")
}

type CreateRequestBody struct {
	Username string `json:"username"`
	Target   int    `json:"target"`
}

func createFarkleHandler(ctx *gin.Context) {
	gameID := uuid.New()
	playerID := uuid.New()

	var requestBody CreateRequestBody
	if err := ctx.ShouldBindJSON(&requestBody); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if requestBody.Target < 1000 || requestBody.Target > 100_000 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Target must be between 1000 and 100000"})
		return
	}

	if len(requestBody.Username) < 3 || len(requestBody.Username) > 20 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Username must be between 3 and 20 characters"})
		return
	}

	createLobby := connect.CreateFarkleRequest{
		GameID: gameID,
		Target: requestBody.Target,
		Player: connect.LobbyPlayer{
			UserID:   playerID,
			Username: requestBody.Username,
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

	accepted, err := createGameFarkle(createLobby)
	if err != nil {
		log.Println("Failed to create game:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create game"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"userId":     playerID,
		"gameId":     accepted.GameID,
		"serverId":   accepted.ServerID,
		"serverHost": accepted.ServerHost,
		"token":      accepted.Auth,
	})
}

type JoinRequestBody struct {
	Username string `json:"username"`
}

func joinFarkleHandler(ctx *gin.Context) {
	gameIDStr := ctx.Param("gameId")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid game ID"})
		return
	}

	var requestBody JoinRequestBody
	if err := ctx.ShouldBindJSON(&requestBody); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(requestBody.Username) < 3 || len(requestBody.Username) > 20 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Username must be between 3 and 20 characters"})
		return
	}

	playerID := uuid.New()
	joinLobby := connect.JoinFarkleRequest{
		Player: connect.LobbyPlayer{
			UserID:   playerID,
			Username: requestBody.Username,
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

	joined, err := joinGameFarkle(gameID, joinLobby)
	if err != nil {
		log.Println("Failed to join game:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to join game"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"userId":     playerID,
		"gameId":     joined.GameID,
		"serverId":   joined.ServerID,
		"serverHost": joined.ServerHost,
		"token":      joined.Auth,
	})
}
