package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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
		UserID:            order.UserID,
		Price:             order.Price,
		Quantity:          order.Quantity,
		Side:              side,
		RemainingQuantity: order.Quantity,
		CreatedAt: pgtype.Timestamp{
			Time:  time.Unix(0, order.CreatedAt),
			Valid: true,
		},
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

func (r *PostgresOrderRepository) UpdateOrderStatusBatch(ctx context.Context, q *db.Queries, makerIDs []uuid.UUID, makerStatuses []domain.OrderStatus) error {
	if len(makerIDs) == 0 {
		return nil
	}

	dbStatuses := make([]string, len(makerStatuses))
	for i, status := range makerStatuses {
		dbStatuses[i] = string(mapStatusToDB(status))
	}

	return q.UpdateOrderStatusesBatch(ctx, db.UpdateOrderStatusesBatchParams{
		Ids:      makerIDs,
		Statuses: dbStatuses,
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
			Quantity:          row.Quantity,
			RemainingQuantity: row.RemainingQuantity,
			CreatedAt:         row.CreatedAt.Time.UnixNano(),
			Side:              side,
		}
	}

	return orders, nil
}
func mapDBToStatus(status db.OrderStatus) domain.OrderStatus {
	switch status {
	case db.OrderStatusPLACED:
		return domain.StatusPlaced
	case db.OrderStatusPARTIAL:
		return domain.StatusPartial
	case db.OrderStatusFILLED:
		return domain.StatusFilled
	case db.OrderStatusCANCELLED:
		return domain.StatusCancelled
	case db.OrderStatusREJECTED:
		return domain.StatusRejected
	default:
		return domain.StatusNew
	}
}

func (r *PostgresOrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	row, err := r.store.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}

	side, err := toDomainSide(row.Side)
	if err != nil {
		return nil, err
	}

	return &domain.Order{
		ID:                row.ID,
		UserID:            row.UserID,
		Price:             row.Price,
		Quantity:          row.Quantity,
		RemainingQuantity: row.RemainingQuantity,
		CreatedAt:         row.CreatedAt.Time.UnixNano(),
		Side:              side,
		Status:            mapDBToStatus(row.Status),
	}, nil
}
