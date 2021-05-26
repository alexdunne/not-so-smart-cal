package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/alexdunne/not-so-smart-cal/weather"
	"github.com/go-redis/redis/v8"
)

var ErrNotFound = errors.New("no results found")

type Storage struct {
	redisClient            *redis.Client
	storageKey             string
	futureEventsStorageKey string
}

func NewStorage(client *redis.Client) *Storage {
	return &Storage{
		redisClient:            client,
		storageKey:             "events",
		futureEventsStorageKey: "events:futureEvents",
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

	return s.convertJsonToEvent(val)
}

func (s *Storage) Set(ctx context.Context, key string, value *weather.Event) error {
	if err := s.storeById(ctx, key, value); err != nil {
		return err
	}

	if value.StartsAt.After(time.Now()) {
		if err := s.storeAsFutureEvent(ctx, value); err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) GetFutureEvents(ctx context.Context, max time.Time) ([]*weather.Event, error) {
	eventIds, err := s.redisClient.ZRangeByScore(ctx, s.futureEventsStorageKey, &redis.ZRangeBy{
		Min: s.now(),
		Max: strconv.FormatInt(max.Unix(), 10),
	}).Result()

	if err == redis.Nil {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	if len(eventIds) == 0 {
		return []*weather.Event{}, nil
	}

	values, err := s.redisClient.HMGet(ctx, s.storageKey, eventIds...).Result()
	if err == redis.Nil {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	results := make([]*weather.Event, 0)
	for _, val := range values {
		if val == nil {
			// reached the end of the values
			break
		}

		event, err := s.convertJsonToEvent(val.(string))
		if err != nil {
			return nil, err
		}

		results = append(results, event)
	}

	return results, nil
}

func (s *Storage) RemoveExpiredFutureEvents(ctx context.Context) error {
	return s.redisClient.ZRemRangeByScore(ctx, s.futureEventsStorageKey, "-inf", s.now()).Err()
}

func (s *Storage) storeById(ctx context.Context, key string, value *weather.Event) error {
	jsonVal, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return s.redisClient.HSet(ctx, s.storageKey, key, string(jsonVal)).Err()
}

func (s *Storage) storeAsFutureEvent(ctx context.Context, value *weather.Event) error {
	if err := s.redisClient.ZRem(ctx, s.futureEventsStorageKey, value.ID).Err(); err != nil {
		return err
	}

	return s.redisClient.ZAdd(
		ctx,
		s.futureEventsStorageKey,
		&redis.Z{Score: float64(value.StartsAt.Unix()), Member: value.ID},
	).Err()
}

func (s *Storage) convertJsonToEvent(jsonVal string) (*weather.Event, error) {
	var result *weather.Event
	if err := json.Unmarshal([]byte(jsonVal), &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Storage) now() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}
