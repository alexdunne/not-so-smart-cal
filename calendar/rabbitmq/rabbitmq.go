package rabbitmq

import (
	"encoding/json"
	"fmt"

	"github.com/streadway/amqp"
)

type Producer struct {
	// Rabbitmq DSN
	connStr string

	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewProducer(connStr string) *Producer {
	return &Producer{
		connStr: connStr,
	}
}

func (p *Producer) Open() (err error) {
	// ensure a DSN is set before attempting to connect.
	if p.connStr == "" {
		return fmt.Errorf("connection string required")
	}

	if p.conn, err = amqp.Dial(p.connStr); err != nil {
		return err
	}

	if p.channel, err = p.conn.Channel(); err != nil {
		return err
	}

	return nil
}

func (p *Producer) Close() {
	p.conn.Close()
	p.channel.Close()
}

func (p *Producer) Publish(eventName string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return p.channel.Publish(
		"calendar",
		eventName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(jsonData),
		})
}
