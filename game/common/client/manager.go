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
	gamesMu   sync.Mutex
	games     map[uuid.UUID]*gameClients
	allClosed chan struct{}
}

func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		gamesMu:   sync.Mutex{},
		games:     make(map[uuid.UUID]*gameClients),
		allClosed: make(chan struct{}),
	}
}

func (manager *WebSocketManager) tryGetGame(gameID uuid.UUID) (*gameClients, bool) {
	manager.gamesMu.Lock()
	defer manager.gamesMu.Unlock()

	game, ok := manager.games[gameID]
	return game, ok
}

func (manager *WebSocketManager) getOrCreateGame(gameID uuid.UUID) *gameClients {
	if game, ok := manager.tryGetGame(gameID); ok {
		return game
	}

	manager.gamesMu.Lock()
	defer manager.gamesMu.Unlock()

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

func (manager *WebSocketManager) UpgradeClient(gameID uuid.UUID, userID uuid.UUID, conn *websocket.Conn) (*WebSocketClient, bool) {
	game, ok := manager.tryGetGame(gameID)
	if !ok {
		return nil, false
	}

	client := game.getOrCreateClient(userID)
	client.reconnect(conn)

	return client, true
}

func (manager *WebSocketManager) CloseGame(gameID uuid.UUID) {
	game := manager.getOrCreateGame(gameID)
	game.close()

	manager.gamesMu.Lock()
	defer manager.gamesMu.Unlock()

	delete(manager.games, gameID)

	if len(manager.games) == 0 {
		close(manager.allClosed)
		manager.allClosed = make(chan struct{})
	}
}

func (manager *WebSocketManager) Close() {
	manager.gamesMu.Lock()
	defer manager.gamesMu.Unlock()

	for _, game := range manager.games {
		game.close()
	}

	manager.games = make(map[uuid.UUID]*gameClients)

	select {
	case <-manager.allClosed:
	default:
		close(manager.allClosed)
	}
}

func (manager *WebSocketManager) AllGamesClosed() <-chan struct{} {
	return manager.allClosed
}
