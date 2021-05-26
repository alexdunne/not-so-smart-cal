package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/alexdunne/not-so-smart-cal/weather"
	"github.com/alexdunne/not-so-smart-cal/weather/openweather"
	weatherRedis "github.com/alexdunne/not-so-smart-cal/weather/redis"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var OPEN_WEATHER_API_KEY = os.Getenv("OPEN_WEATHER_API_KEY")

type WeatherService interface {
	FetchWeather(location *weather.GeocodedLocation, timeToCheckFor time.Time) (*weather.WeatherSummary, error)
}

type EventStorage interface {
	GetFutureEvents(ctx context.Context, max time.Time) ([]*weather.Event, error)
	Set(ctx context.Context, eventId string, value *weather.Event) error
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

	minutes := flag.Int("minutes", 1440, "how many minutes in the future should we check")
	workers := flag.Int("workers", 3, "how many workers should be created")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("error creating the logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
	})
	defer redisClient.Close()
	logger.Info("opened redis connection")

	weatherService := openweather.NewWeatherService(logger, OPEN_WEATHER_API_KEY)
	eventStorage := weatherRedis.NewStorage(redisClient)

	go func() {
		// do this is the background as we're not really bothered about the results
		logger.Info("removing expired future events")
		eventStorage.RemoveExpiredFutureEvents(ctx)
	}()

	logger.Info("fetching future events", zap.Int("minutes", *minutes))

	events, err := eventStorage.GetFutureEvents(ctx, time.Now().Add(time.Hour*time.Duration(*minutes)))
	if err != nil {
		logger.Error("error fetching future events", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("fetched future events", zap.Int("eventsCount", len(events)))

	if len(events) == 0 {
		// no events found, we're done here
		os.Exit(0)
	}

	pending, complete := make(chan *weather.Event), make(chan string)

	// kick off the workers
	for i := 0; i < int(*workers); i++ {
		w := &Worker{
			id:             uuid.NewString(),
			logger:         logger,
			eventStorage:   eventStorage,
			weatherService: weatherService,
		}

		go w.Run(pending, complete)
	}

	logger.Info("populating workers with future events")
	go func() {
		for _, event := range events {
			pending <- event
		}
	}()

	go func() {
		completed := 0
		for {
			<-complete

			completed++
			logger.Info("event finished processing", zap.Int("completed", completed))

			if completed == len(events) {
				logger.Info("all events finished processing", zap.Int("completed", completed))
				ctx.Done()
			}
		}
	}()

	<-ctx.Done()

	close(pending)
	close(complete)

	os.Exit(0)
}

type Worker struct {
	id             string
	logger         *zap.Logger
	eventStorage   EventStorage
	weatherService WeatherService
}

func (w *Worker) Run(in <-chan *weather.Event, out chan<- string) {
	w.logger.Info("starting worker", zap.String("workerId", w.id))

	for event := range in {
		w.logger.Info("starting to process event", zap.String("workerId", w.id), zap.String("eventId", event.ID))

		weatherResponse, err := w.weatherService.FetchWeather(event.GeocodedLocation, event.StartsAt)
		if err != nil {
			w.logger.Error("error whilst fetching weather data", zap.String("workerId", w.id), zap.Error(err))
			continue
		}

		w.logger.Info(
			"weather fetched for event",
			zap.String("workerId", w.id),
			zap.String("eventId", event.ID),
			zap.Any("weather", weatherResponse),
		)

		w.logger.Info("caching event weather", zap.String("workerId", w.id), zap.String("eventId", event.ID))

		w.eventStorage.Set(context.Background(), event.ID, &weather.Event{
			ID:               event.ID,
			StartsAt:         event.StartsAt,
			GeocodedLocation: event.GeocodedLocation,
			WeatherSummary:   weatherResponse,
		})

		w.logger.Info("finished processing event", zap.String("workerId", w.id), zap.String("eventId", event.ID))
		out <- event.ID
	}
}
