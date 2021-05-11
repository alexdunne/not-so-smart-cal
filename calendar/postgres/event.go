package postgres

import (
	"context"

	"github.com/alexdunne/not-so-smart-cal/calendar/model"
	"github.com/go-playground/validator/v10"
)

type EventService struct {
	DB        *DB
	Validator *validator.Validate
}

func (s *EventService) CreateEvent(ctx context.Context, event *model.Event) error {
	tx, err := s.DB.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	event.CreatedAt = s.DB.now()

	err = s.Validator.Struct(event)
	if err != nil {
		return err.(validator.ValidationErrors)
	}

	var id string
	err = tx.QueryRow(ctx, `
			INSERT INTO events (title, location, starts_at, ends_at, created_at)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING "id"
		`,
		event.Title,
		event.Location,
		event.StartsAt,
		event.EndsAt,
		event.CreatedAt,
	).Scan(&id)
	if err != nil {
		return err
	}

	event.ID = id

	return tx.Commit(ctx)
}
