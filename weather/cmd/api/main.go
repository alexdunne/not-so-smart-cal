package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/alexdunne/not-so-smart-cal/weather"
	weatherRedis "github.com/alexdunne/not-so-smart-cal/weather/redis"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type EventWeatherStorage interface {
	Get(ctx context.Context, eventId string) (*weather.WeatherSummary, error)
	Set(ctx context.Context, eventId string, value *weather.WeatherSummary) error
}

func main() {
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

	eventWeatherStorage := weatherRedis.NewEventWeatherStorage(redisClient)

	r := gin.Default()

	r.GET("/event/:eventId", func(c *gin.Context) {
		eventId := c.Param("eventId")

		summary, err := eventWeatherStorage.Get(c.Request.Context(), eventId)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{})
			return
		}

		c.JSON(200, gin.H{
			"data": gin.H{
				"weather": summary,
			},
		})
	})

	r.Run()
}
