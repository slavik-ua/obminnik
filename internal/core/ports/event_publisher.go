package ports

import (
	"context"
	"simple-orderbook/internal/core/domain"
)

type EventPublisher interface {
	Publish(ctx context.Context, event *domain.OutboxEvent) error
}
