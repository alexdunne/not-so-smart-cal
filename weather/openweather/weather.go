package openweather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/alexdunne/not-so-smart-cal/weather"
	"go.uber.org/zap"
)

type WeatherService struct {
	logger            *zap.Logger
	openWeatherAPIKey string
}

func NewWeatherService(logger *zap.Logger, openWeatherAPIKey string) *WeatherService {
	return &WeatherService{
		logger:            logger,
		openWeatherAPIKey: openWeatherAPIKey,
	}
}

func (ws *WeatherService) FetchWeather(
	location *weather.GeocodedLocation,
	timeToCheckFor time.Time,
) (*weather.WeatherSummary, error) {
	checkForTimeTruncatedToHour := time.Date(
		timeToCheckFor.Year(), timeToCheckFor.Month(), timeToCheckFor.Day(),
		timeToCheckFor.Hour(), 0, 0, 0, timeToCheckFor.Location(),
	)

	hoursUntilTime := time.Until(checkForTimeTruncatedToHour).Hours()

	switch {
	case hoursUntilTime < 48:
		return ws.fetchWeatherFromHourlyForecast(location, timeToCheckFor)

	case hoursUntilTime < 168:
		return ws.fetchWeatherFromDailyForecast(location, timeToCheckFor)
	}

	return nil, fmt.Errorf("event too far in the future")

}

func (ws *WeatherService) fetchWeatherFromHourlyForecast(
	location *weather.GeocodedLocation,
	timeToCheckFor time.Time,
) (*weather.WeatherSummary, error) {
	result, err := ws.fetchWeatherForLocation(location)
	if err != nil {
		return nil, err
	}

	// now find the weather for the relevant hour
	for _, item := range result.Hourly {
		if int64(item.Dt) >= timeToCheckFor.Unix() {
			weatherInfo := item.Weather[0]

			return &weather.WeatherSummary{
				Type:        weatherInfo.Main,
				Description: weatherInfo.Description,
				Temp:        fmt.Sprintf("%f", item.Temp),
			}, nil
		}
	}

	return nil, fmt.Errorf("weather for time after %d was not found", timeToCheckFor.Unix())
}

func (ws *WeatherService) fetchWeatherFromDailyForecast(
	location *weather.GeocodedLocation,
	timeToCheckFor time.Time,
) (*weather.WeatherSummary, error) {

	result, err := ws.fetchWeatherForLocation(location)
	if err != nil {
		return nil, err
	}

	// now find the weather for the relevant day
	for _, item := range result.Daily {
		if int64(item.Dt) >= timeToCheckFor.Unix() {
			weatherInfo := item.Weather[0]

			var temp float64

			if timeToCheckFor.Hour() < 10 {
				temp = item.Temp.Morn
			} else if timeToCheckFor.Hour() < 17 {
				temp = item.Temp.Day
			} else if timeToCheckFor.Hour() < 20 {
				temp = item.Temp.Eve
			} else {
				temp = item.Temp.Night
			}

			return &weather.WeatherSummary{
				Type:        weatherInfo.Main,
				Description: weatherInfo.Description,
				Temp:        fmt.Sprintf("%f", temp),
			}, nil
		}
	}

	return nil, fmt.Errorf("weather for time after %d was not found", timeToCheckFor.Unix())
}

func (ws *WeatherService) fetchWeatherForLocation(
	location *weather.GeocodedLocation,
) (*weather.WeatherResponse, error) {
	url := fmt.Sprintf(
		"https://api.openweathermap.org/data/2.5/onecall?lat=%s&lon=%s&exclude=current,minutely,alerts&units=metric&appid=%s",
		location.Latitude,
		location.Longitude,
		ws.openWeatherAPIKey,
	)

	ws.logger.Debug("requesting weather information", zap.String("url", url))

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response *weather.WeatherResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response, nil
}
