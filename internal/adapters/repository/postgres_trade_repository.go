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

func (tr *PostgresTradeRepository) CreateBatch(ctx context.Context, q *db.Queries, trades []domain.Trade, orderSide domain.OrderSide) error {
	if len(trades) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, len(trades))
	buyers := make([]uuid.UUID, len(trades))
	sellers := make([]uuid.UUID, len(trades))
	takers := make([]uuid.UUID, len(trades))
	makers := make([]uuid.UUID, len(trades))
	prices := make([]int64, len(trades))
	qtys := make([]int64, len(trades))

	for i, t := range trades {
		ids[i] = t.ID
		takers[i] = t.TakerUserID
		makers[i] = t.MakerUserID
		prices[i] = t.Price
		qtys[i] = t.Quantity

		if orderSide == domain.SideBuy {
			buyers[i] = t.TakerOrderID
			sellers[i] = t.MakerOrderID
		} else {
			buyers[i] = t.MakerOrderID
			sellers[i] = t.TakerOrderID
		}
	}

	return q.CreateTradesBatch(ctx, db.CreateTradesBatchParams{
		Ids:             ids,
		BuyerOrderIds:   buyers,
		SellerOrderIds:  sellers,
		TakerUserIds:    takers,
		MakerUserIds:    makers,
		ExecutionPrices: prices,
		Quantities:      qtys,
	})
}
