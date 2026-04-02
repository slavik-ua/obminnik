package ports

import (
	"context"

	"simple-orderbook/internal/db"
	"simple-orderbook/internal/core/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, q *db.Queries, order *domain.Order) error
}
