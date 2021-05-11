package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alexdunne/not-so-smart-cal/calendar/model"
	"github.com/alexdunne/not-so-smart-cal/calendar/postgres"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func main() {
	fmt.Println("calendar service booting")

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
	)

	db := postgres.NewDB(connStr)
	if err := db.Open(context.Background()); err != nil {
		fmt.Printf("cannot open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close(context.Background())

	validate = validator.New()

	eventService := &postgres.EventService{
		DB:        db,
		Validator: validate,
	}

	server := &Server{
		eventService: eventService,
	}

	r := gin.Default()

	r.POST("/event", server.createEvent)

	r.Run()
}

type Server struct {
	eventService *postgres.EventService
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
