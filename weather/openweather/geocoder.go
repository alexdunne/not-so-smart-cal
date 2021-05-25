package openweather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alexdunne/not-so-smart-cal/weather"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type GeocodeService struct {
	storage           Storage
	logger            *zap.Logger
	openWeatherAPIKey string
}

func NewGeocodeService(redisClient *redis.Client, logger *zap.Logger, openWeatherAPIKey string) *GeocodeService {
	return &GeocodeService{
		storage:           &Cache{redisClient: redisClient, storageKey: "geocoded_locations"},
		logger:            logger,
		openWeatherAPIKey: openWeatherAPIKey,
	}
}

type geocodedResponseItem struct {
	Name      string      `json:"name"`
	Latitude  json.Number `json:"lat"`
	Longitude json.Number `json:"lon"`
	Country   string      `json:"country"`
}

// GeocodeLocation attempts to geocode a given location string
func (gs *GeocodeService) GeocodeLocation(ctx context.Context, location string) (*weather.GeocodedLocation, error) {
	if result, err := gs.storage.Get(ctx, location); err != nil {
		if !errors.Is(err, ErrNotFound) {
			gs.logger.Error("error whilst attempting to get location for storage", zap.Error(err))
		}
	} else {
		return result, nil
	}

	url := fmt.Sprintf(
		"http://api.openweathermap.org/geo/1.0/direct?q=%s&limit=1&appid=%s",
		location,
		gs.openWeatherAPIKey,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var locations []*geocodedResponseItem

	if err := json.NewDecoder(resp.Body).Decode(&locations); err != nil {
		return nil, err
	}

	if len(locations) == 0 {
		return nil, fmt.Errorf("no geocoded locations could be found for %s", location)
	}

	geocodedLocation := &weather.GeocodedLocation{
		Name:      locations[0].Name,
		Latitude:  string(locations[0].Latitude),
		Longitude: string(locations[0].Longitude),
		Country:   locations[0].Country,
	}

	if err := gs.storage.Set(ctx, location, geocodedLocation); err != nil {
		gs.logger.Error("error whilst attempting to save location for storage", zap.Error(err))
	}

	return geocodedLocation, nil
}

type Storage interface {
	Get(ctx context.Context, key string) (*weather.GeocodedLocation, error)
	Set(ctx context.Context, key string, value *weather.GeocodedLocation) error
}

var ErrNotFound = errors.New("no results found")

// todo - use a LRU with a capacity instead
type Cache struct {
	redisClient *redis.Client
	storageKey  string
}

func (c *Cache) Get(ctx context.Context, key string) (*weather.GeocodedLocation, error) {
	val, err := c.redisClient.HGet(ctx, c.storageKey, key).Result()

	if err == redis.Nil {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var result *weather.GeocodedLocation
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	return result, err
}

func (c *Cache) Set(ctx context.Context, key string, value *weather.GeocodedLocation) error {
	jsonVal, err := json.Marshal(value)
	if err != nil {
		return err
	}

	if err := c.redisClient.HSet(ctx, c.storageKey, key, string(jsonVal)).Err(); err != nil {
		return err
	}

	return nil
}
