package ports

import (
	"context"

	"simple-orderbook/internal/core/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
}
