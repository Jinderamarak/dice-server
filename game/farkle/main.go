package main

import (
	"context"
	"dice-server/common/queue"
	"dice-server/common/telemetry"
	"dice-server/common/utility"
	"dice-server/game/common/client"
	"dice-server/game/farkle/internal/config"
	"dice-server/game/farkle/internal/lobby"
	"dice-server/game/farkle/internal/web"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Panicln("Failed to load configuration:", err)
	}

	log.Println("Starting server with ID:", config.Config.Server.ID)

	otelShutdown, err := telemetry.Setup(context.Background())
	if err != nil {
		log.Panicln("Failed to initialize OpenTelemetry:", err)
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			log.Println("OTel shutdown error:", err)
		}
	}()

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

	ctx := setupGracefulShutdown()

	workerResult := make(chan error)
	go lobby.RunWorker(workerResult, pool, manager, ctx)

	webResult := make(chan error)
	entry := web.NewEntryPoint(manager)
	go entry.Run(webResult)

	select {
	case err := <-workerResult:
		log.Println("Worker loop failed:", err)
	case err := <-webResult:
		log.Println("Web server failed:", err)
	}
}

func setupGracefulShutdown() context.Context {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := <-sigs
		log.Println("Received shutdown signal:", sig)

		cancel()
	}()

	return ctx
}
