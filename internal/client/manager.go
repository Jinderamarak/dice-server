package client

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

type ClientManager struct {
	upgrader websocket.Upgrader
	closed   chan uuid.UUID

	clientsMu sync.Mutex
	clients   map[uuid.UUID]*WebSocketClient
}

func NewClientsManager(upgrader websocket.Upgrader) *ClientManager {
	return &ClientManager{
		upgrader: upgrader,
		clients:  make(map[uuid.UUID]*WebSocketClient),
		closed:   make(chan uuid.UUID),
	}
}

func (manager *ClientManager) ClosureHandler() {
	for session := range manager.closed {
		log.Println("Removing session:", session)
		manager.clientsMu.Lock()
		delete(manager.clients, session)
		manager.clientsMu.Unlock()
	}
}

func (manager *ClientManager) Upgrade(session uuid.UUID, ctx *gin.Context) (*WebSocketClient, bool, error) {
	log.Println("Upgrading connection with session:", session)

	conn, err := manager.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return nil, false, err
	}

	var client *WebSocketClient
	var reconnected bool

	{
		manager.clientsMu.Lock()
		client, reconnected = manager.clients[session]
		if !reconnected {
			client = newClient(conn, session, manager.closed)
			manager.clients[session] = client
		}
		manager.clientsMu.Unlock()
	}

	if reconnected {
		log.Println("Reconnected session, notifying client")
		if err := client.NotifyReconnection(conn); err != nil {
			return nil, false, err
		}
	}

	log.Println("Upgrade finished")
	return client, reconnected, nil
}
