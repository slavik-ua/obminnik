package services

import (
	"context"
	"log"
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
		publisher: publisher,
	}
}

func (r *OutboxRelay) Run(ctx context.Context) {
	for {
		err := r.process(ctx)
		backoff := 100 * time.Millisecond
		if err != nil {
			backoff = 1 * time.Second
		}

		select {
		case <-ctx.Done():
			log.Println("outbox relay shutting down")
			return
		case <-time.After(backoff):
		}
	}
}

func (r *OutboxRelay) process(ctx context.Context) error {
	events, err := r.outboxRepo.FetchUnprocessed(ctx, 10)
	if err != nil {
		log.Printf("outbox relay: fetch failed: %v", err)
		return err
	}

	for _, event := range events {
		if err := r.publisher.Publish(ctx, event); err != nil {
			log.Printf("outbox relay: publish failed for event %s: %v", event.ID, err)
			continue
		}

		if err := r.outboxRepo.MarkProcessed(ctx, event.ID); err != nil {
			log.Printf("outbox relay: mark processed failed for event %s: %v", event.ID, err)
		}
	}

	return nil
}