package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/alexdunne/not-so-smart-cal/weather"
	"github.com/go-redis/redis/v8"
)

var ErrNotFound = errors.New("no results found")

type Storage struct {
	redisClient *redis.Client
	storageKey  string
}

func NewStorage(client *redis.Client) *Storage {
	return &Storage{
		redisClient: client,
		storageKey:  "events",
	}
}

func (s *Storage) Get(ctx context.Context, key string) (*weather.Event, error) {
	val, err := s.redisClient.HGet(ctx, s.storageKey, key).Result()

	fmt.Printf("%v", val)

	if err == redis.Nil {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var result *weather.Event
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	return result, err
}

func (s *Storage) Set(ctx context.Context, key string, value *weather.Event) error {
	jsonVal, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if err := s.redisClient.HSet(ctx, s.storageKey, key, string(jsonVal)).Err(); err != nil {
		return err
	}

	return nil
}
