package client

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"sync"
)

type ClientManager struct {
	upgrader   websocket.Upgrader
	terminated chan uuid.UUID

	clientsMu sync.Mutex
	clients   map[uuid.UUID]*WebSocketClient
}

func NewClientsManager(upgrader websocket.Upgrader) *ClientManager {
	return &ClientManager{
		upgrader:   upgrader,
		clients:    make(map[uuid.UUID]*WebSocketClient),
		terminated: make(chan uuid.UUID),
	}
}

func (manager *ClientManager) TerminationHandler() {
	for session := range manager.terminated {
		manager.clientsMu.Lock()
		delete(manager.clients, session)
		manager.clientsMu.Unlock()
	}
}

func (manager *ClientManager) Upgrade(session uuid.UUID, ctx *gin.Context) (*WebSocketClient, bool, error) {
	conn, err := manager.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return nil, false, err
	}

	var client *WebSocketClient
	var reconnected bool
	{
		manager.clientsMu.Lock()
		defer manager.clientsMu.Unlock()

		client, reconnected = manager.clients[session]
		if !reconnected {
			client = newClient(conn, session, manager.terminated)
			manager.clients[session] = client
		}
	}

	if reconnected {
		client.NotifyReconnection(conn)
	}

	return client, reconnected, nil
}
