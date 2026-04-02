package ports

import (
	"context"

	"simple-orderbook/internal/core/domain"
)

type TradeRepository interface {
	CreateTrade(ctx context.Context, trade *domain.Trade) error
}