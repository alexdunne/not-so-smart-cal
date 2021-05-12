package main

import (
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"
)

func main() {
	err := run()

	if err != nil {
		log.Fatalf("worker error: %v", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	connStr := fmt.Sprintf(
		"amqp://%s:%s@%s:%s",
		os.Getenv("AMQP_USER"),
		os.Getenv("AMQP_PASSWORD"),
		os.Getenv("AMQP_HOST"),
		os.Getenv("AMQP_PORT"),
	)

	conn, err := amqp.Dial(connStr)
	if err != nil {
		return err
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	msgs, err := channel.Consume(
		"fetch_event_weather", // queue
		"",                    // consumer
		true,                  // auto-ack
		false,                 // exclusive
		false,                 // no-local
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		return err
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("received a message: %s", d.Body)
		}
	}()

	log.Printf("waiting for messages")
	<-forever

	return nil
}
