package rabbitmq

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type CalendarPublisher struct {
	conn         *amqp.Connection
	channel      *amqp.Channel
	exchangeName string
	logger       *zap.Logger
}

func NewCalendarPublisher(
	conn *amqp.Connection,
	exchangeName string,
	logger *zap.Logger,
) (*CalendarPublisher, error) {
	amqpChan, err := conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "error creating channel")
	}

	return &CalendarPublisher{
		conn:         conn,
		channel:      amqpChan,
		exchangeName: exchangeName,
		logger:       logger,
	}, nil
}

func (p *CalendarPublisher) Setup() error {
	p.logger.Info("configuring exchange")

	err := p.channel.ExchangeDeclare(p.exchangeName, "topic", true, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "error creating the exchange")
	}

	return nil
}

func (p *CalendarPublisher) Close() error {
	if err := p.channel.Close(); err != nil {
		return errors.Wrap(err, "error closing the publisher channel")
	}

	if err := p.conn.Close(); err != nil {
		return errors.Wrap(err, "error closing the publisher connection")
	}

	return nil
}

func (p *CalendarPublisher) Publish(routingKey string, data interface{}) error {
	p.logger.Info("publishing message", zap.String("exchange", p.exchangeName), zap.String("routing key", routingKey))

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return p.channel.Publish(
		p.exchangeName,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			MessageId:   uuid.New().String(),
			Timestamp:   time.Now(),
			Body:        []byte(jsonData),
		})
}
