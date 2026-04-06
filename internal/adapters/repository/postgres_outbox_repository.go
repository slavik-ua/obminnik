package repository

import (
	"context"

	"github.com/google/uuid"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"
)

type PostgresOutboxRepository struct {
	store *db.Store
}

func NewPostgresOutboxRepository(store *db.Store) *PostgresOutboxRepository {
	return &PostgresOutboxRepository{store: store}
}

func (r *PostgresOutboxRepository) AddEvent(ctx context.Context, q *db.Queries, event *domain.OutboxEvent) error {
	return q.AddOutboxEvent(ctx, db.AddOutboxEventParams{
		ID:      event.ID,
		Type:    event.Type,
		Payload: event.Payload,
	})
}

func (r *PostgresOutboxRepository) FetchUnprocessed(ctx context.Context, limit int32) ([]*domain.OutboxEvent, error) {
	rows, err := r.store.FetchUnprocessedEvents(ctx, limit)
	if err != nil {
		return nil, err
	}

	events := make([]*domain.OutboxEvent, len(rows))
	for i, row := range rows {
		events[i] = &domain.OutboxEvent{
			ID:          row.ID,
			Type:        row.Type,
			Payload:     row.Payload,
			CreatedAt:   row.CreatedAt.Time,
			ProcessedAt: row.ProcessedAt.Time,
		}
	}

	return events, nil
}

func (r *PostgresOutboxRepository) MarkProcessed(ctx context.Context, id uuid.UUID) error {
	return r.store.MarkEventProcessed(ctx, id)
}
