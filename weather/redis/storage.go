package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alexdunne/not-so-smart-cal/weather"
	"github.com/go-redis/redis/v8"
)

var ErrNotFound = errors.New("no results found")

type EventWeatherStorage struct {
	redisClient *redis.Client
	keyPrefix   string
}

func NewEventWeatherStorage(client *redis.Client) *EventWeatherStorage {
	return &EventWeatherStorage{
		redisClient: client,
		keyPrefix:   "event",
	}
}

func (s *EventWeatherStorage) Get(ctx context.Context, key string) (*weather.WeatherSummary, error) {
	val, err := s.redisClient.Get(ctx, s.key(key)).Result()

	if err == redis.Nil {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var result *weather.WeatherSummary
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	return result, err
}

func (s *EventWeatherStorage) Set(ctx context.Context, key string, value *weather.WeatherSummary) error {
	val, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if err := s.redisClient.Set(ctx, s.key(key), string(val), 1*time.Hour).Err(); err != nil {
		return err
	}

	return nil
}

func (s *EventWeatherStorage) key(key string) string {
	return fmt.Sprintf("%s:%s", s.keyPrefix, key)
}
