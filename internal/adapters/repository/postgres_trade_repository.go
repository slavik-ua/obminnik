package repository

import (
	"context"

	"github.com/google/uuid"

	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"
)

type PostgresTradeRepository struct {
	store *db.Store
}

func NewPostgresTradeRepository(store *db.Store) *PostgresTradeRepository {
	return &PostgresTradeRepository{
		store: store,
	}
}

func (tr *PostgresTradeRepository) Create(ctx context.Context, q *db.Queries, trade *domain.Trade, buyerID, sellerID uuid.UUID) error {
	params := db.CreateTradeParams{
		ID:             trade.ID,
		BuyerOrderID:   buyerID,
		SellerOrderID:  sellerID,
		TakerUserID:    trade.TakerUserID,
		MakerUserID:    trade.MakerUserID,
		ExecutionPrice: trade.Price,
		Quantity:       trade.Quantity,
	}

	_, err := q.CreateTrade(ctx, params)
	return err
}
