package ports

import (
	"context"

	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"
)

type TradeRepository interface {
	Create(ctx context.Context, q *db.Queries, trade *domain.Trade) error
}
