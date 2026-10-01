package store

import (
	"context"
	"fmt"
	"time"

	"github.com/PithomLabs/ebp/internal/domain"
)

// EventRepository handles append-only event log persistence.
type EventRepository struct{}

// NewEventRepository creates a new EventRepository.
func NewEventRepository() *EventRepository {
	return &EventRepository{}
}

// Append inserts an event into the append-only log.
func (r *EventRepository) Append(ctx context.Context, q Querier, event *domain.Event) error {
	if event.ID == "" {
		event.ID = generateUUID()
	}
	if event.CreatedAt == "" {
		event.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := q.ExecContext(ctx,
		`INSERT INTO ebp_event (id, idea_id, action, detail, actor, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.ID, event.IdeaID, event.Action, event.Detail, event.Actor, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

// ListByIdea returns all events for an idea, ordered by creation time.
func (r *EventRepository) ListByIdea(ctx context.Context, q Querier, ideaID string) ([]domain.Event, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, idea_id, action, detail, actor, created_at
		 FROM ebp_event WHERE idea_id = ? ORDER BY created_at`, ideaID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []domain.Event
	for rows.Next() {
		var e domain.Event
		if err := rows.Scan(&e.ID, &e.IdeaID, &e.Action, &e.Detail, &e.Actor, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
