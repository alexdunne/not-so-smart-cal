package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alexdunne/not-so-smart-cal/calendar/postgres"
	"github.com/gin-gonic/gin"
)

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

	fmt.Println(connStr)

	db := postgres.NewDB(connStr)
	if err := db.Open(context.Background()); err != nil {
		fmt.Printf("cannot open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close(context.Background())

	r := gin.Default()

	r.Run()
}
