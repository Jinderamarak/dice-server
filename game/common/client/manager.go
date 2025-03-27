package client

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"sync"
)

type gameClients struct {
	mu      sync.Mutex
	clients map[uuid.UUID]*WebSocketClient
}

func (clients *gameClients) getOrCreateClient(userID uuid.UUID) *WebSocketClient {
	clients.mu.Lock()
	defer clients.mu.Unlock()

	if client, ok := clients.clients[userID]; ok {
		return client
	}

	client := NewWebSocketClient()
	clients.clients[userID] = client
	return client
}

func (clients *gameClients) close() {
	clients.mu.Lock()
	defer clients.mu.Unlock()

	for _, client := range clients.clients {
		client.close()
	}
}

type WebSocketManager struct {
	upgrader websocket.Upgrader

	gamesMu sync.Mutex
	games   map[uuid.UUID]*gameClients
}

func NewWebSocketManager(upgrader websocket.Upgrader) *WebSocketManager {
	return &WebSocketManager{
		upgrader: upgrader,
		gamesMu:  sync.Mutex{},
		games:    make(map[uuid.UUID]*gameClients),
	}
}

func (manager *WebSocketManager) getOrCreateGame(gameID uuid.UUID) *gameClients {
	manager.gamesMu.Lock()
	defer manager.gamesMu.Unlock()

	if game, ok := manager.games[gameID]; ok {
		return game
	}

	game := &gameClients{
		mu:      sync.Mutex{},
		clients: make(map[uuid.UUID]*WebSocketClient),
	}

	manager.games[gameID] = game
	return game
}

func (manager *WebSocketManager) GetClient(gameID uuid.UUID, userID uuid.UUID) *WebSocketClient {
	game := manager.getOrCreateGame(gameID)
	client := game.getOrCreateClient(userID)
	return client
}

func (manager *WebSocketManager) UpgradeClient(gameID uuid.UUID, userID uuid.UUID, conn *websocket.Conn) *WebSocketClient {
	game := manager.getOrCreateGame(gameID)
	client := game.getOrCreateClient(userID)

	client.reconnect(conn)

	return client
}

func (manager *WebSocketManager) CloseGame(gameID uuid.UUID) {
	game := manager.getOrCreateGame(gameID)
	game.close()

	manager.gamesMu.Lock()
	defer manager.gamesMu.Unlock()

	delete(manager.games, gameID)
}
