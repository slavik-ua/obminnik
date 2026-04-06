package services

import (
	"context"
	"log/slog"
	"time"

	"simple-orderbook/internal/core/ports"
)

type OutboxRelay struct {
	outboxRepo ports.OutboxRepository
	publisher  ports.EventPublisher
}

func NewOutboxRelay(outboxRepo ports.OutboxRepository, publisher ports.EventPublisher) *OutboxRelay {
	return &OutboxRelay{
		outboxRepo: outboxRepo,
		publisher:  publisher,
	}
}

func (r *OutboxRelay) Run(ctx context.Context) {
	slog.Info("outbox relay started")
	for {
		err := r.process(ctx)
		backoff := 100 * time.Millisecond
		if err != nil {
			backoff = 1 * time.Second
		}

		select {
		case <-ctx.Done():
			slog.Info("outbox relay shutting down")
			return
		case <-time.After(backoff):
		}
	}
}

func (r *OutboxRelay) process(ctx context.Context) error {
	events, err := r.outboxRepo.FetchUnprocessed(ctx, 10)
	if err != nil {
		slog.Error("outbox relay: fetch failed", "error", err)
		return err
	}

	for _, event := range events {
		if err := r.publisher.Publish(ctx, event); err != nil {
			slog.Error("outbox relay: publish failed",
				"event_id", event.ID,
				"error", err,
			)
			continue
		}

		if err := r.outboxRepo.MarkProcessed(ctx, event.ID); err != nil {
			slog.Error("outbox relay: mark processed failed",
				"event_id", event.ID,
				"error", err,
			)
		}
	}

	return nil
}
