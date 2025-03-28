package main

import (
	"dice-server/common/queue"
	"dice-server/common/utility"
	"dice-server/game/common/client"
	"dice-server/game/farkle/internal/lobby"
	"dice-server/game/farkle/internal/web"
	"log"
)

func main() {
	pool, err := queue.NewPool(8, "amqp://guest:guest@localhost:5672/")
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
