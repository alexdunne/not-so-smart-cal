package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alexdunne/not-so-smart-cal/calendar/model"
	"github.com/alexdunne/not-so-smart-cal/calendar/postgres"
	"github.com/alexdunne/not-so-smart-cal/calendar/rabbitmq"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func main() {
	fmt.Println("calendar service booting")

	validate = validator.New()

	dbConnStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)

	db := postgres.NewDB(dbConnStr)
	if err := db.Open(context.Background()); err != nil {
		fmt.Printf("cannot open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close(context.Background())

	amqpConnStr := fmt.Sprintf(
		"amqp://%s:%s@%s:%s",
		os.Getenv("AMQP_USER"),
		os.Getenv("AMQP_PASSWORD"),
		os.Getenv("AMQP_HOST"),
		os.Getenv("AMQP_PORT"),
	)

	producer := rabbitmq.NewProducer(amqpConnStr)
	if err := producer.Open(); err != nil {
		fmt.Printf("cannot open rabbitmq connection: %v\n", err)
		os.Exit(1)
	}
	defer producer.Close()

	eventService := &postgres.EventService{
		DB:        db,
		Producer:  producer,
		Validator: validate,
	}

	server := &Server{
		eventService: eventService,
	}

	r := gin.Default()

	r.GET("/event/:eventId", server.findEvent)
	r.POST("/event", server.createEvent)

	r.Run()
}

type Server struct {
	eventService *postgres.EventService
}

func (s *Server) findEvent(c *gin.Context) {
	eventId := c.Param("eventId")

	event, err := s.eventService.FindEventByID(c.Request.Context(), eventId)
	if err != nil {
		ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"event": event,
	}})
}

type CreateEventInput struct {
	Title    string    `json:"title" binding:"required,min=2"`
	Location string    `json:"location"`
	StartsAt time.Time `json:"startsAt" binding:"required" time_format:"2006-01-02T15:04:05Z07:00"`
	EndsAt   time.Time `json:"endsAt" binding:"required,gtfield=StartsAt" time_format:"2006-01-02T15:04:05Z07:00"`
}

func (s *Server) createEvent(c *gin.Context) {
	var input CreateEventInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event := &model.Event{
		Title:    input.Title,
		Location: input.Location,
		StartsAt: input.StartsAt,
		EndsAt:   input.EndsAt,
	}

	err := s.eventService.CreateEvent(c.Request.Context(), event)

	if err != nil {
		ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"event": event,
	}})
}

func ErrorResponse(c *gin.Context, err error) {
	// Log this error
	fmt.Printf("error response: %v\n", err)

	// Provide a better way for the application to return a appropriate statis code and message
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
