package ports

import (
	"context"
	"simple-orderbook/internal/core/domain"
)

type OrderService interface {
	PlaceOrder(ctx context.Context, order *domain.Order) ([]domain.Trade, error)
}