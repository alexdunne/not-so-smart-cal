package weather

import "time"

type Event struct {
	ID               string            `json:"id"`
	StartsAt         time.Time         `json:"startsAt"`
	GeocodedLocation *GeocodedLocation `json:"geocodedLocation"`
	WeatherSummary   *WeatherSummary   `json:"weatherSummary"`
}

type GeocodedLocation struct {
	Name      string
	Latitude  string
	Longitude string
	Country   string
}

type WeatherSummary struct {
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
