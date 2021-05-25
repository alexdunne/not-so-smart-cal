package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/alexdunne/not-so-smart-cal/weather"
	"github.com/alexdunne/not-so-smart-cal/weather/openweather"
	weatherRedis "github.com/alexdunne/not-so-smart-cal/weather/redis"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

var OPEN_WEATHER_API_KEY = os.Getenv("OPEN_WEATHER_API_KEY")

type GeocodeService interface {
	GeocodeLocation(ctx context.Context, location string) (*weather.GeocodedLocation, error)
}

type WeatherService interface {
	FetchWeather(location *weather.GeocodedLocation, timeToCheckFor time.Time) (*weather.WeatherSummary, error)
}

type EventWeatherStorage interface {
	Get(ctx context.Context, eventId string) (*weather.WeatherSummary, error)
	Set(ctx context.Context, eventId string, value *weather.WeatherSummary) error
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// setup signal handlers
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		<-signalCh
		cancel()
	}()

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("error creating the logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	connStr := fmt.Sprintf(
		"amqp://%s:%s@%s:%s",
		os.Getenv("AMQP_USER"),
		os.Getenv("AMQP_PASSWORD"),
		os.Getenv("AMQP_HOST"),
		os.Getenv("AMQP_PORT"),
	)
	amqpConn, err := amqp.Dial(connStr)
	if err != nil {
		logger.Fatal("error opening rabbitmq connection", zap.Error(err))
		os.Exit(1)
	}
	defer amqpConn.Close()
	logger.Info("opened rabbitmq connection")

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	})
	defer redisClient.Close()
	logger.Info("opened redis connection")

	geocoder := openweather.NewGeocodeService(redisClient, logger, OPEN_WEATHER_API_KEY)
	weatherService := openweather.NewWeatherService(logger, OPEN_WEATHER_API_KEY)
	eventWeatherStorage := weatherRedis.NewEventWeatherStorage(redisClient)

	consumer := NewCalendarEventWeatherConsumer(
		amqpConn,
		geocoder,
		weatherService,
		eventWeatherStorage,
		logger,
	)

	go func() {
		logger.Info("starting CalendarEventWeather consumer")
		err := consumer.StartConsumer("calendar", "event.created", "fetch_weather_for_event")
		if err != nil {
			logger.Fatal("error whilst running consumer", zap.Error(err))
			cancel()
		}
	}()

	// wait for termination
	<-ctx.Done()
}

type CalendarEventWeatherConsumer struct {
	conn                *amqp.Connection
	geocoder            GeocodeService
	weatherService      WeatherService
	eventWeatherStorage EventWeatherStorage
	logger              *zap.Logger
}

func NewCalendarEventWeatherConsumer(
	conn *amqp.Connection,
	geocoder GeocodeService,
	weatherService WeatherService,
	eventWeatherStorage EventWeatherStorage,
	logger *zap.Logger,
) *CalendarEventWeatherConsumer {
	return &CalendarEventWeatherConsumer{
		conn:                conn,
		geocoder:            geocoder,
		weatherService:      weatherService,
		eventWeatherStorage: eventWeatherStorage,
		logger:              logger,
	}
}

func (c *CalendarEventWeatherConsumer) StartConsumer(exchangeName, routingKey, queueName string) error {
	ch, err := c.createChannel(exchangeName, routingKey, queueName)
	if err != nil {
		return errors.Wrap(err, "error creating channel")
	}
	defer ch.Close()

	messages, err := ch.Consume(queueName, "", true, false, false, false, nil)
	if err != nil {
		return errors.Wrap(err, "error whilst consuming messages")
	}

	c.logger.Info("starting workers")
	// kick off a worker to proccess the incoming messages
	go c.worker(messages)

	chanErr := <-ch.NotifyClose(make(chan *amqp.Error))

	c.logger.Info("channel notified to close")

	return chanErr
}

// createChannel creates a channel from the amqp connection
// and creates all of the necessary exchanges, queues, and bindings
func (c *CalendarEventWeatherConsumer) createChannel(
	exchangeName, routingKey, queueName string,
) (*amqp.Channel, error) {
	ch, err := c.conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "error creating amqp channel")
	}

	err = ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating the exchange")
	}

	queue, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating the queue")
	}

	err = ch.QueueBind(queue.Name, routingKey, exchangeName, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "error binding queue to exchange")
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		return nil, errors.Wrap(err, "error configuring prefetch")
	}

	return ch, nil
}

func (c *CalendarEventWeatherConsumer) worker(messages <-chan amqp.Delivery) {
	for delivery := range messages {
		ctx := context.Background()

		c.logger.Info("received a message")

		var event Event
		err := json.Unmarshal(delivery.Body, &event)

		if err != nil {
			c.logger.Error("error unmarshaling event", zap.Error(err))
			return
		}

		c.logger.Info("starting to process event", zap.String("eventId", event.ID), zap.Any("event", event))

		if time.Until(event.StartsAt).Hours() <= 0 {
			c.logger.Info("not fetching weather information for events in the past")
			continue
		}

		location, err := c.geocoder.GeocodeLocation(ctx, event.Location)
		if err != nil {
			c.logger.Error("error whilst fetching location", zap.Error(err))
			continue
		}

		c.logger.Info(
			"event location geocoded",
			zap.String("location", event.Location),
			zap.String("lat", location.Latitude),
			zap.String("lon", location.Longitude),
		)

		weather, err := c.weatherService.FetchWeather(location, event.StartsAt)
		if err != nil {
			c.logger.Error("error whilst fetching weather data", zap.Error(err))
			continue
		}

		c.logger.Info("weather fetched for event", zap.String("eventId", event.ID), zap.Any("weather", weather))

		c.logger.Info("caching event weather", zap.String("eventId", event.ID))
		c.eventWeatherStorage.Set(ctx, event.ID, weather)
	}
}

type Event struct {
	ID       string    `json:"id"`
	Location string    `json:"location"`
	StartsAt time.Time `json:"startsAt"`
}
