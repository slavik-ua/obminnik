package ports

import (
	"context"

	"github.com/google/uuid"

	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"
)

type OrderRepository interface {
	Create(ctx context.Context, q *db.Queries, order *domain.Order) error
	Cancel(ctx context.Context, q *db.Queries, id uuid.UUID) error
	ListActiveBySide(ctx context.Context, side db.OrderSide) ([]*domain.Order, error)
}
