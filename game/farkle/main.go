package main

import (
	"dice-server/common/utility"
	"dice-server/game/farkle/connect"
	"dice-server/game/farkle/internal/lobby"
	"encoding/json"
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Panicf("Queue connection failed: %s", err)
	}
	defer utility.CloseAndIgnore(conn)

	err = workerLoop(conn)
	if err != nil {
		log.Panicf("Worker loop failed: %s", err)
	}
}

func workerLoop(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		log.Println("Failed to open a channel")
		return err
	}
	defer utility.CloseAndIgnore(ch)

	q, err := ch.QueueDeclare(
		connect.CreateLobbyQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println("Failed to declare a queue")
		return err
	}

	if err = ch.Qos(
		1,
		0,
		false,
	); err != nil {
		log.Println("Failed to set QoS")
		return err
	}

	messages, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Println("Failed to register a consumer")
		return err
	}

	log.Println("Starting worker loop")
	for msg := range messages {
		err = startLobby(conn, msg)
		if err != nil {
			log.Println("Failed to start lobby:", err)
		} else {
			err = msg.Ack(false)
			if err != nil {
				log.Println("Failed to ack message:", err)
			}
		}
	}

	return errors.New("worker ran out of messages")
}

func startLobby(conn *amqp.Connection, msg amqp.Delivery) error {
	var createLobby connect.CreateLobbyMessage
	err := json.Unmarshal(msg.Body, &createLobby)
	if err != nil {
		log.Println("Failed to unmarshal create message:", err)
		return err
	}

	log.Println("Creating lobby for game:", createLobby.GameId)
	go lobby.RunLobby(conn, &createLobby)
	return nil
}
