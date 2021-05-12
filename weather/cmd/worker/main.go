package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

var OPEN_WEATHER_API_KEY = os.Getenv("OPEN_WEATHER_API_KEY")

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

	logger.Info("opened rabbitmq connection")

	consumer := NewCalendarEventWeatherConsumer(amqpConn, logger)

	go func() {
		logger.Info("starting CalendarEventWeather consumer")
		err := consumer.StartConsumer("calendar", "event.created", "enrich_event_with_weather")
		if err != nil {
			logger.Fatal("error whilst running consumer", zap.Error(err))
			cancel()
		}
	}()

	// wait for termination
	<-ctx.Done()
}

type CalendarEventWeatherConsumer struct {
	conn   *amqp.Connection
	logger *zap.Logger
}

func NewCalendarEventWeatherConsumer(conn *amqp.Connection, logger *zap.Logger) *CalendarEventWeatherConsumer {
	return &CalendarEventWeatherConsumer{
		conn:   conn,
		logger: logger,
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
		c.logger.Info("received a message")

		var event Event
		err := json.Unmarshal(delivery.Body, &event)

		if err != nil {
			c.logger.Error("error unmarshaling event", zap.Error(err))
			return
		}

		c.logger.Info("Starting to process for event", zap.String("eventId", event.ID), zap.Any("event", event))

		if time.Until(event.StartsAt).Hours() <= 0 {
			c.logger.Info("not fetching weather information for events in the past")
			continue
		}

		location, err := geocodeLocation(event.Location)
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

		weather, err := fetchWeatherForLocation(*location, event.StartsAt)
		if err != nil {
			c.logger.Error("error whilst fetching weather data", zap.Error(err))
			continue
		}

		c.logger.Info("weather fetched for event", zap.Any("weather", weather))

		// todo - Send this back to the calendar service
	}
}

type Event struct {
	ID       string    `json:"id"`
	Location string    `json:"location"`
	StartsAt time.Time `json:"startsAt"`
}

type GeocodedResponseItem struct {
	Name      string      `json:"name"`
	Latitude  json.Number `json:"lat"`
	Longitude json.Number `json:"lon"`
	Country   string      `json:"country"`
}

type GeocodedLocation struct {
	Name      string
	Latitude  string
	Longitude string
	Country   string
}

func geocodeLocation(location string) (*GeocodedLocation, error) {
	// todo - cache the results of this in a LRU cache
	url := fmt.Sprintf(
		"http://api.openweathermap.org/geo/1.0/direct?q=%s&limit=1&appid=%s",
		location,
		OPEN_WEATHER_API_KEY,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var locations []*GeocodedResponseItem

	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		return nil, err
	}

	if len(locations) == 0 {
		return nil, fmt.Errorf("no geocoded locations could be found for %s", location)
	}

	return &GeocodedLocation{
		Name:      locations[0].Name,
		Latitude:  string(locations[0].Latitude),
		Longitude: string(locations[0].Longitude),
		Country:   locations[0].Country,
	}, nil
}

type Weather struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Temp        string `json:"temp"`
}

type WeatherResponse struct {
	Hourly []WeatherHourlyResponse `json:"hourly"`
	Daily  []WeatherDailyResponse  `json:"daily"`
}

type WeatherHourlyResponse struct {
	Dt        int                  `json:"dt"`
	Temp      float64              `json:"temp"`
	FeelsLike float64              `json:"feels_like"`
	Weather   []WeatherInformation `json:"weather"`
}

type WeatherDailyResponse struct {
	Dt   int `json:"dt"`
	Temp struct {
		Day   float64 `json:"day"`
		Night float64 `json:"night"`
		Eve   float64 `json:"eve"`
		Morn  float64 `json:"morn"`
	} `json:"temp"`
	Weather []WeatherInformation `json:"weather"`
}

type WeatherInformation struct {
	ID          int    `json:"id"`
	Main        string `json:"main"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func fetchWeatherForLocation(location GeocodedLocation, checkFor time.Time) (*Weather, error) {
	// floor the checkFor date to the hour
	checkForHour := time.Date(
		checkFor.Year(), checkFor.Month(), checkFor.Day(),
		checkFor.Hour(), 0, 0, 0, checkFor.Location(),
	)

	hoursUntilTime := time.Until(checkForHour).Hours()

	switch {
	case hoursUntilTime < 48:
		hourlyWeather, err := fetchHourlyWeather(location)
		if err != nil {
			return nil, err
		}

		// now find the weather for the relevant hour
		for _, item := range hourlyWeather {
			if int64(item.Dt) == checkForHour.Unix() {
				weatherInfo := item.Weather[0]

				return &Weather{
					Type:        weatherInfo.Main,
					Description: weatherInfo.Description,
					Temp:        fmt.Sprintf("%f", item.Temp),
				}, nil
			}
		}

		return nil, fmt.Errorf("hour matching %d was not found", checkForHour.Unix())

	case hoursUntilTime < 168:
		dailyWeather, err := fetchDalyWeather(location)
		if err != nil {
			return nil, err
		}

		// floor the checkFor date to the day
		checkForMidday := time.Date(
			checkFor.Year(), checkFor.Month(), checkFor.Day(),
			12, 0, 0, 0, checkFor.Location(),
		)

		// now find the weather for the relevant day
		for _, item := range dailyWeather {
			if int64(item.Dt) == checkForMidday.Unix() {
				weatherInfo := item.Weather[0]

				var temp float64

				if checkForHour.Hour() < 10 {
					temp = item.Temp.Morn
				} else if checkForHour.Hour() < 17 {
					temp = item.Temp.Day
				} else if checkForHour.Hour() < 20 {
					temp = item.Temp.Eve
				} else {
					temp = item.Temp.Night
				}

				return &Weather{
					Type:        weatherInfo.Main,
					Description: weatherInfo.Description,
					Temp:        fmt.Sprintf("%f", temp),
				}, nil
			}
		}

		return nil, fmt.Errorf("day matching %d was not found", checkForMidday.Unix())
	}

	return nil, fmt.Errorf("event too far in the future")
}

func fetchHourlyWeather(location GeocodedLocation) ([]WeatherHourlyResponse, error) {
	url := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/onecall?lat=%s&lon=%s&exclude=current,minutely,daily,alerts&appid=%s",
		location.Latitude,
		location.Longitude,
		OPEN_WEATHER_API_KEY,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response *WeatherResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Hourly, nil
}

func fetchDalyWeather(location GeocodedLocation) ([]WeatherDailyResponse, error) {
	url := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/onecall?lat=%s&lon=%s&exclude=current,minutely,hourly,alerts&appid=%s",
		location.Latitude,
		location.Longitude,
		OPEN_WEATHER_API_KEY,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response *WeatherResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Daily, nil
}
