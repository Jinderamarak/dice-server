package lobby

import (
	"context"
	"dice-server/common/auth/token"
	"dice-server/common/queue"
	"dice-server/common/telemetry"
	"dice-server/game/common/client"
	"dice-server/game/farkle/connect"
	"dice-server/game/farkle/internal/config"
	"encoding/json"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"log"
	"time"
)

var shutdownTimeout = time.Minute * 30

func RunWorker(fail chan<- error, pool *queue.Pool, manager *client.WebSocketManager, ctx context.Context) {
	err := workerLoop(pool, manager, ctx)
	log.Println("Worker loop closing:", err)

	closed := manager.TryWaitForAllClosed(shutdownTimeout)
	if !closed {
		err = errors.Wrap(err, "game shutdown timeout")
	} else {
		log.Println("All games shutdown")
	}

	fail <- err
}

func workerLoop(pool *queue.Pool, manager *client.WebSocketManager, ctx context.Context) error {
	consumer := pool.GetConsumer(connect.CreateLobbyQueue)
	defer consumer.Close()

	messages := consumer.Consume()
	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return errors.New("worker consumer channel closed")
			}

			err := attemptLobby(pool, manager, &msg)
			if err != nil {
				if err = msg.Nack(false, false); err != nil {
					return errors.Wrap(err, "failed to nack message")
				}
			} else {
				if err = msg.Ack(false); err != nil {
					return errors.Wrap(err, "failed to ack message")
				}
			}
		case <-ctx.Done():
			log.Println("Worker consumer shutting down")
			return errors.New("worker consumer shutting down")
		}
	}
}

func attemptLobby(pool *queue.Pool, manager *client.WebSocketManager, msg *amqp.Delivery) error {
	ctx := telemetry.ExtractAMQP(context.Background(), msg)

	var createLobby connect.CreateFarkleRequest
	if err := json.Unmarshal(msg.Body, &createLobby); err != nil {
		return errors.Wrap(err, "failed to unmarshal create lobby message")
	}

	go startLobby(ctx, pool, manager, &createLobby)
	return nil
}

func startLobby(ctx context.Context, pool *queue.Pool, manager *client.WebSocketManager, msg *connect.CreateFarkleRequest) {
	tracer := otel.Tracer("game-farkle")
	ctx, span := tracer.Start(ctx, "startLobby",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attribute.String("game.id", msg.GameID.String())),
	)
	defer span.End()

	auth := token.NewGameToken(msg.Player.UserID, msg.GameID, config.Config.Server.ID, config.Config.Server.Host, config.Config.Auth.Issuer, time.Now())
	authToken, err := auth.Sign([]byte(config.Config.Auth.Secret))
	if err != nil {
		span.RecordError(err)
		log.Println("Failed to sign auth token:", err)
		return
	}

	publisher := pool.GetPublisher(connect.AcceptLobbyQueue(msg.GameID))
	err = publisher.PublishJSON(ctx, connect.CreateFarkleResponse{
		GameID:     msg.GameID,
		ServerID:   config.Config.Server.ID,
		ServerHost: config.Config.Server.Host,
		Auth:       authToken,
	})
	if err != nil {
		span.RecordError(err)
		log.Println("Failed to publish acceptation message:", err)
		return
	}
	publisher.Close()

	err = runLobby(ctx, pool, manager, msg)
	if err != nil {
		span.RecordError(err)
		log.Println("Failed to run lobby:", err)
	}
}
