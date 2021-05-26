package postgres

import (
	"context"
	"time"

	"github.com/alexdunne/not-so-smart-cal/calendar/model"
	"github.com/alexdunne/not-so-smart-cal/calendar/rabbitmq"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type EventService struct {
	DB                *DB
	CalendarPublisher *rabbitmq.CalendarPublisher
	Validator         *validator.Validate
	Logger            *zap.Logger
}

func (s *EventService) FindInTimeRange(ctx context.Context, startsAt, endsAt time.Time) ([]*model.Event, error) {
	s.Logger.Info(
		"Finding events in time range",
		zap.String("startsAt", startsAt.Format(time.RFC3339)),
		zap.String("endsAt", endsAt.Format(time.RFC3339)),
	)

	tx, err := s.DB.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, title, location, starts_at, ends_at, created_at 
		FROM events
		WHERE ends_at >= $1 AND starts_at <= $2
	`, startsAt.Format(time.RFC3339), endsAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*model.Event, 0)
	for rows.Next() {
		var event model.Event
		if err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.Location,
			&event.StartsAt,
			&event.EndsAt,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}

		events = append(events, &event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (s *EventService) FindEventByID(ctx context.Context, id string) (*model.Event, error) {
	tx, err := s.DB.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	event := &model.Event{}

	err = tx.QueryRow(ctx, `
		SELECT id, title, location, starts_at, ends_at, created_at
		FROM events
		WHERE id = $1
	`, id).Scan(
		&event.ID,
		&event.Title,
		&event.Location,
		&event.StartsAt,
		&event.EndsAt,
		&event.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return event, nil
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

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	s.CalendarPublisher.Publish("event.created", event)

	return nil
}
