package ports

import (
	"context"

	"github.com/google/uuid"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"
)

type OutboxRepository interface {
	AddEvent(ctx context.Context, q *db.Queries, event *domain.OutboxEvent) error
	FetchUnprocessed(ctx context.Context, limit int32) ([]*domain.OutboxEvent, error)
	MarkProcessed(ctx context.Context, id uuid.UUID) error
}
