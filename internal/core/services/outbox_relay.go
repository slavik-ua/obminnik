package services

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"simple-orderbook/internal/core/ports"
)

type OutboxRelay struct {
	outboxRepo ports.OutboxRepository
	publisher  ports.EventPublisher
	notify     chan struct{}
}

func NewOutboxRelay(outboxRepo ports.OutboxRepository, publisher ports.EventPublisher) *OutboxRelay {
	return &OutboxRelay{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		notify:     make(chan struct{}, 1),
	}
}

func (r *OutboxRelay) Notify() {
	select {
	case r.notify <- struct{}{}:
	default:
	}
}

func (r *OutboxRelay) Run(ctx context.Context) {
	slog.Info("outbox relay started")
	for {
		processedCount, err := r.process(ctx)
		if err == nil && processedCount > 0 {
			continue
		}

		select {
		case <-ctx.Done():
			slog.Info("outbox relay shutting down")
			return
		case <-r.notify:
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func (r *OutboxRelay) process(ctx context.Context) (int, error) {
	events, err := r.outboxRepo.FetchUnprocessed(ctx, 100)
	if err != nil {
		slog.Error("outbox relay: fetch failed", "error", err)
		return 0, err
	}

	if len(events) == 0 {
		return 0, nil
	}

	processedIDs := make([]uuid.UUID, 0, len(events))

	for _, event := range events {
		if err := r.publisher.Publish(ctx, event); err != nil {
			slog.Error("outbox relay: publish failed",
				"event_id", event.ID,
				"error", err,
			)
			continue
		}

		processedIDs = append(processedIDs, event.ID)
	}

	if len(processedIDs) > 0 {
		if err := r.outboxRepo.MarkProcessedBatch(ctx, processedIDs); err != nil {
			slog.Error("outbox relay: batch mark failed",
				"error", err,
			)
			return len(processedIDs), err
		}
	}

	return len(processedIDs), nil
}
