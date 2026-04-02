package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx"
	"simple-orderbook/internal/db"
	"simple-orderbook/internal/core/domain"
)

type PostgresTradeRepository struct {
	store *db.Store
}

func NewPostgresTradeRepository(store *db.Store) *PostgresTradeRepository {
	return &PostgresTradeRepository{
		store: store,
	}
}

func (tr *PostgresTradeRepository) Create(ctx context.Context, q *db.Queries, trade *domain.Trade) error {
	params := db.CreateTradeParams{
		ID:             trade.ID,
		BuyerOrderID:   trade.MakerOrderID,
		SellerOrderID:  trade.TakerOrderID,
		TakerUserID:    trade.TakerUserID,
		MakerUserID:    trade.MakerUserID,
		ExecutionPrice: trade.Price,
		Quantity:       trade.Quantity,
	}

	_, err := q.CreateTrade(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	return nil
}