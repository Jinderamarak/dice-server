package main

import (
	"dice-server/common/queue"
	"dice-server/common/utility"
	"dice-server/common/web"
	"dice-server/game-master/internal/config"
	"dice-server/game/farkle/connect"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"time"
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

	mux := http.NewServeMux()
	mux.HandleFunc("/api/game/farkle", corsed(createFarkleHandler))
	mux.HandleFunc("/api/game/farkle/{gameId}/join", corsed(joinFarkleHandler))

	addr := fmt.Sprintf(":%d", config.Config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
		//	Short timeouts since requests are quickly upgraded
		ReadTimeout:       time.Second * 10,
		ReadHeaderTimeout: time.Second * 10,
		WriteTimeout:      time.Second * 10,
		IdleTimeout:       time.Second * 10,
		MaxHeaderBytes:    1 << 20,
	}

	if err = server.ListenAndServe(); err != nil {
		log.Panicln("Failed to run server:", err)
	}
}

func corsed(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		fn(w, r)
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

func createFarkleHandler(w http.ResponseWriter, r *http.Request) {
	gameID := uuid.New()
	playerID := uuid.New()

	var requestBody CreateRequestBody
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		web.SendJson(w, http.StatusBadRequest, web.D{"error": "Invalid request body"})
	}

	if requestBody.Target < 1000 || requestBody.Target > 100_000 {
		web.SendJson(w, http.StatusBadRequest, web.D{"error": "Target must be between 1000 and 100000"})
		return
	}

	if len(requestBody.Username) < 3 || len(requestBody.Username) > 20 {
		web.SendJson(w, http.StatusBadRequest, web.D{"error": "Username must be between 3 and 20 characters"})
		return
	}

	createLobby := connect.CreateFarkleRequest{
		GameID: gameID,
		Target: requestBody.Target,
		Player: connect.LobbyPlayer{
			UserID:   playerID,
			Username: requestBody.Username,
			DiceSet: defaultDiceSet(),
		},
	}

	accepted, err := createGameFarkle(createLobby)
	if err != nil {
		log.Println("Failed to create game:", err)
		web.SendJson(w, http.StatusInternalServerError, web.D{"error": "Failed to create game"})
		return
	}

	web.SendJson(w, http.StatusCreated, web.D{
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

func joinFarkleHandler(w http.ResponseWriter, r *http.Request) {
	gameIDStr := r.PathValue("gameId")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		web.SendJson(w, http.StatusBadRequest, web.D{"error": "Invalid game ID"})
		return
	}

	var requestBody JoinRequestBody
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		web.SendJson(w, http.StatusBadRequest, web.D{"error": "Invalid request body"})
		return
	}

	if len(requestBody.Username) < 3 || len(requestBody.Username) > 20 {
		web.SendJson(w, http.StatusBadRequest, web.D{"error": "Username must be between 3 and 20 characters"})
		return
	}

	playerID := uuid.New()
	joinLobby := connect.JoinFarkleRequest{
		Player: connect.LobbyPlayer{
			UserID:   playerID,
			Username: requestBody.Username,
			DiceSet: defaultDiceSet(),
		},
	}

	joined, err := joinGameFarkle(gameID, joinLobby)
	if err != nil {
		log.Println("Failed to join game:", err)
		web.SendJson(w, http.StatusInternalServerError, web.D{"error": "Failed to join game"})
		return
	}

	web.SendJson(w, http.StatusOK, web.D{
		"userId":     playerID,
		"gameId":     joined.GameID,
		"serverId":   joined.ServerID,
		"serverHost": joined.ServerHost,
		"token":      joined.Auth,
	})
}

func defaultDiceSet() []connect.LobbyDice {
	dice := make([]connect.LobbyDice, 6)
	for i := range dice {
		dice[i] = connect.LobbyDice{ID: uuid.New(), Variant: connect.DiceRegular}
	}
	return dice
}

