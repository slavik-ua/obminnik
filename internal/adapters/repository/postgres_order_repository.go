package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

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

func (r *PostgresOrderRepository) Create(ctx context.Context, q *db.Queries, order *domain.Order) error {
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

func (r *PostgresOrderRepository) Cancel(ctx context.Context, q *db.Queries, id uuid.UUID) error {
	return q.CancelOrder(ctx, id)
}

func toDomainSide(side db.OrderSide) (domain.OrderSide, error) {
	switch side {
	case db.OrderSideBUY:
		return domain.SideBuy, nil
	case db.OrderSideSELL:
		return domain.SideSell, nil
	default:
		return 0, fmt.Errorf("invalid db order side: %v", side)
	}
}

func mapStatusToDB(status domain.OrderStatus) db.OrderStatus {
	switch status {
	case domain.StatusPlaced:
		return db.OrderStatusPLACED
	case domain.StatusPartial:
		return db.OrderStatusPARTIAL
	case domain.StatusFilled:
		return db.OrderStatusFILLED
	case domain.StatusCancelled:
		return db.OrderStatusCANCELLED
	default:
		return db.OrderStatusNEW
	}
}

func (r *PostgresOrderRepository) UpdateStatus(ctx context.Context, q *db.Queries, id uuid.UUID, status domain.OrderStatus) error {
	return q.UpdateOrderStatus(ctx, db.UpdateOrderStatusParams{
		ID:     id,
		Status: mapStatusToDB(status),
	})
}

func (r *PostgresOrderRepository) ListActiveBySide(ctx context.Context, side db.OrderSide) ([]*domain.Order, error) {
	rows, err := r.store.ListActiveOrdersBySide(ctx, side)
	if err != nil {
		return nil, err
	}

	orders := make([]*domain.Order, len(rows))
	for i, row := range rows {
		side, err := toDomainSide(row.Side)
		if err != nil {
			return nil, err
		}
		orders[i] = &domain.Order{
			ID:                row.ID,
			UserID:            row.UserID,
			Price:             row.Price,
			CreatedAt:         row.CreatedAt.Time.UnixNano(),
			RemainingQuantity: row.RemainingQuantity,
			Side:              side,
		}
	}

	return orders, nil
}
