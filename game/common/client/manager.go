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

	clientsMu sync.Mutex
	clients   map[uuid.UUID]*gameClients
}

func NewWebSocketManager(upgrader websocket.Upgrader) *WebSocketManager {
	return &WebSocketManager{
		upgrader:  upgrader,
		clientsMu: sync.Mutex{},
		clients:   make(map[uuid.UUID]*gameClients),
	}
}

func (manager *WebSocketManager) getOrCreateGame(gameID uuid.UUID) *gameClients {
	manager.clientsMu.Lock()
	defer manager.clientsMu.Unlock()

	if clients, ok := manager.clients[gameID]; ok {
		return clients
	}

	clients := &gameClients{
		mu:      sync.Mutex{},
		clients: make(map[uuid.UUID]*WebSocketClient),
	}

	manager.clients[gameID] = clients
	return clients
}

func (manager *WebSocketManager) UpgradeClient(gameID uuid.UUID, userID uuid.UUID, conn *websocket.Conn) *WebSocketClient {
	clients := manager.getOrCreateGame(gameID)
	client := clients.getOrCreateClient(userID)

	client.reconnect(conn)

	return client
}

func (manager *WebSocketManager) CloseGame(gameID uuid.UUID) {
	clients := manager.getOrCreateGame(gameID)
	clients.close()

	manager.clientsMu.Lock()
	defer manager.clientsMu.Unlock()

	delete(manager.clients, gameID)
}
