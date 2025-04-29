package web

import (
	"dice-server/common/auth/token"
	"dice-server/common/web"
	"dice-server/game/common/client"
	"dice-server/game/farkle/internal/config"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type EntryPoint struct {
	manager *client.WebSocketManager
}

func NewEntryPoint(manager *client.WebSocketManager) *EntryPoint {
	return &EntryPoint{manager: manager}
}

func (entry *EntryPoint) Run(fail chan<- error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/game/farkle/{auth}", corsed(entry.entryHandler))

	addr := fmt.Sprintf(":%d", config.Config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
		//	Short timeouts since requests are quickly upgraded
		ReadTimeout:       time.Second * 10,
		ReadHeaderTimeout: time.Second * 10,
		WriteTimeout:      time.Second * 10,
		IdleTimeout:       time.Second * 10,
		MaxHeaderBytes:    10 << 10,
	}

	err := server.ListenAndServe()
	fail <- err
}

func corsed(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		fn(w, r)
	}
}

func (entry *EntryPoint) entryHandler(w http.ResponseWriter, r *http.Request) {
	auth := r.PathValue("auth")
	gameToken, err := token.ValidateGameToken(auth, []byte(config.Config.Auth.Secret))
	if err != nil {
		web.SendJson(w, http.StatusUnauthorized, web.D{"error": "Invalid token"})
		return
	}

	if gameToken.ServerID != config.Config.Server.ID {
		web.SendJson(w, http.StatusUnauthorized, web.D{"error": "Invalid server ID"})
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		web.SendJson(w, http.StatusUnauthorized, web.D{"error": "Failed to upgrade connection"})
		return
	}

	_, ok := entry.manager.UpgradeClient(gameToken.GameID, gameToken.UserID, conn)
	if !ok {
		web.SendJson(w, http.StatusNotFound, web.D{"error": "Game not found"})
		return
	}
}
