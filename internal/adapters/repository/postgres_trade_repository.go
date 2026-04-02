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

func NewPostgreSTradeRepository(store *db.Store) *PostgresTradeRepository {
	return &PostgresTradeRepository{
		store: store,
	}
}

func (tr *PostgresTradeRepository) CreateTrade(ctx context.Context, trade *domain.Trade) error {
	params := db.CreateTradeParams{
		ID:             trade.ID,
		BuyerOrderID:   trade.MakerOrderID,
		SellerOrderID:  trade.TakerOrderID,
		TakerUserID:    trade.TakerUserID,
		MakerUserID:    trade.MakerUserID,
		ExecutionPrice: trade.Price,
		Quantity:       trade.Quantity,
	}

	_, err := tr.store.CreateTrade(ctx, params)
	if err != nil {
		// the trade already exists
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return err
	}

	return nil
}