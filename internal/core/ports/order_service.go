package ports

import (
	"context"
	"simple-orderbook/internal/core/domain"

	"github.com/google/uuid"
)

type OrderService interface {
	PlaceOrder(ctx context.Context, order *domain.Order) error
	CancelOrder(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
	GetOrderBook(ctx context.Context) ([]byte, error)
	Deposit(ctx context.Context, userID uuid.UUID, asset string, amount int64) error
	GetBalances(ctx context.Context, userID uuid.UUID) ([]domain.BalanceRecord, error)
}
