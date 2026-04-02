package repository

import (
	"context"
	"fmt"
	
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/db"
)

type PostgresOrderRepository struct {
	store *db.Store
}

func NewPostgresOrderRepository(store *db.Store) *PostgresOrderRepository {
	return &PostgresOrderRepository{
		store: store,
	}
}

func toDBSide(order domain.OrderSide) (db.OrderSide, error) {
	switch order {
	case domain.SideBuy:
		return db.OrderSideBUY, nil
	case domain.SideSell:
		return db.OrderSideSELL, nil
	default:
		return "", fmt.Errorf("invalid order side: %v", order)
	}
}

func (pso *PostgresOrderRepository) Create(ctx context.Context, q *db.Queries, order *domain.Order) error {
	side, err := toDBSide(order.Side)
	if err != nil {
		return err
	}

	params := db.CreateOrderParams{
		ID:                order.ID,
		Price:             order.Price,
		Quantity:          order.Quantity,
		Side:              side,
		RemainingQuantity: order.Quantity,
	}

	_, err = q.CreateOrder(ctx, params)
	return err
}
