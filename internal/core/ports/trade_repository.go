package ports

import (
	"context"

	"simple-orderbook/internal/db"
	"simple-orderbook/internal/core/domain"
)

type TradeRepository interface {
	Create(ctx context.Context, q *db.Queries, trade *domain.Trade) error
}