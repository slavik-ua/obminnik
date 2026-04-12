package ports

import (
	"context"
	"simple-orderbook/internal/core/domain"

	"github.com/google/uuid"
)

type OrderService interface {
	PlaceOrder(ctx context.Context, order *domain.Order) error
	CancelOrder(ctx context.Context, id uuid.UUID) error
	GetOrderBook(ctx context.Context) ([]byte, error)
}
