package services

import (
	"context"

	"simple-orderbook/internal/core/ports"
	"simple-orderbook/internal/db"
	"simple-orderbook/internal/core/domain"
)

type OrderService struct {
	orderRepo ports.OrderRepository
	tradeRepo ports.TradeRepository
	store     *db.Store
	book      *domain.OrderBook
}

func NewOrderService(store *db.Store, orderRepo ports.OrderRepository, tradeRepo ports.TradeRepository, book *domain.OrderBook) *OrderService {
	return &OrderService{
		store:     store,
		orderRepo: orderRepo,
		tradeRepo: tradeRepo,
		book:      book,
	}
}

func (s *OrderService) PlaceOrder(ctx context.Context, order *domain.Order) ([]domain.Trade, error) {
	trades := s.book.PlaceOrder(order.ID, order.UserID, order.Price, order.Quantity, order.Side, nil)

	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := s.orderRepo.Create(ctx, q, order); err != nil {
			return err
		}

		for i := range trades {
			if err := s.tradeRepo.Create(ctx, q, &trades[i]); err != nil {
				return err
			}
		}

		return nil
	})

	return trades, err
}