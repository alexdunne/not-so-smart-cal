package model

import "time"

type Event struct {
	ID        string    `json:"id"`
	Title     string    `json:"title" validate:"required,min=2"`
	Location  string    `json:"location"`
	StartsAt  time.Time `json:"startsAt" validate:"required"`
	EndsAt    time.Time `json:"endsAt" validate:"required"`
	CreatedAt time.Time `json:"createdAt" validate:"required"`
}
