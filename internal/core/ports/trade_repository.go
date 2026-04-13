package ports

import (
	"context"

	"github.com/google/uuid"

	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"
)

type TradeRepository interface {
	Create(ctx context.Context, q *db.Queries, trade *domain.Trade, buyerID, sellerID uuid.UUID) error
}
