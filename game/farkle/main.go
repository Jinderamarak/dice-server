package main

import (
	"dice-server/common/queue"
	"dice-server/common/utility"
	"dice-server/game/common/client"
	"dice-server/game/farkle/internal/config"
	"dice-server/game/farkle/internal/lobby"
	"dice-server/game/farkle/internal/web"
	"log"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Panicln("Failed to load configuration:", err)
	}

	log.Println("Starting server with ID:", config.Config.Server.ID)

	pool, err := queue.NewPool(
		config.Config.Rabbit.Connections,
		config.Config.Rabbit.Channels,
		config.Config.Rabbit.URL,
	)
	if err != nil {
		log.Panicln("Queue pool creation failed:", err)
	}
	defer utility.CloseAndIgnore(pool)

	manager := client.NewWebSocketManager()
	defer manager.Close()

	workerResult := make(chan error)
	go lobby.RunWorker(workerResult, pool, manager)

	webResult := make(chan error)
	entry := web.NewEntryPoint(manager)
	go entry.Run(webResult)

	select {
	case err := <-workerResult:
		log.Panicln("Worker loop failed:", err)
	case err := <-webResult:
		log.Panicln("Web server failed:", err)
	}
}
