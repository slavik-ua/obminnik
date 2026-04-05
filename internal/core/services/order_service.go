package services

import (
	"context"
	"fmt"
	"encoding/json"
	"log"

	"github.com/google/uuid"

	"simple-orderbook/internal/core/ports"
	"simple-orderbook/internal/db"
	"simple-orderbook/internal/core/domain"
)

type OrderService struct {
	orderRepo ports.OrderRepository
	tradeRepo ports.TradeRepository
	outboxRepo ports.OutboxRepository
	store     *db.Store
	book      *domain.OrderBook
	cache     ports.OrderBookCache
}

func NewOrderService(store *db.Store, orderRepo ports.OrderRepository, tradeRepo ports.TradeRepository, outboxRepo ports.OutboxRepository, book *domain.OrderBook, cache ports.OrderBookCache) *OrderService {
	return &OrderService{
		store:      store,
		orderRepo:  orderRepo,
		tradeRepo:  tradeRepo,
		outboxRepo: outboxRepo,
		book:       book,
		cache:      cache,
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

			payload, err := json.Marshal(trades[i])
			if err != nil {
				return fmt.Errorf("marshal trade event: %w", err)
			}

			event := &domain.OutboxEvent{
				ID:      uuid.New(),
				Type:    "TradeExecuted",
				Payload: payload,
			}
			if err := s.outboxRepo.AddEvent(ctx, q, event); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := s.cache.Invalidate(ctx); err != nil {
		log.Printf("cache invalidate failed after PlaceOrder: %v", err)
	}

	return trades, err
}

func (s *OrderService) CancelOrder(ctx context.Context, id uuid.UUID) error {
	cancelled := s.book.CancelOrder(id)
	if !cancelled {
		return fmt.Errorf("order not found: %s", id)
	}

	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		return s.orderRepo.Cancel(ctx, q, id)
	})

	if err != nil {
		return err
	}

	if err := s.cache.Invalidate(ctx); err != nil {
		log.Printf("cache invalidation failed after CancelOrder: %v", err)
	}

	return nil
}

func (s *OrderService) GetOrderBook(ctx context.Context) ([]byte, error) {
	data, found, err := s.cache.Get(ctx)
	if err != nil {
		// fail-open
		log.Printf("cache got error: %v", err)
	}
	if found {
		return data, nil
	}

	snapshot := s.book.Snapshot()
	data, err = json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal orderbook: %w", err)
	}

	if err := s.cache.Set(ctx, data); err != nil {
		log.Printf("cache set error: %v", err)
	}

	return data, nil
}

func (s *OrderService) RebuildOrderBook(ctx context.Context) error {
	for _, side := range []db.OrderSide{db.OrderSideBUY, db.OrderSideSELL} {
		orders, err := s.orderRepo.ListActiveBySide(ctx, side)
		if err != nil {
			return fmt.Errorf("rebuild orderbook: %w", err)
		}
		for _, o := range orders {
			s.book.RestoreOrder(o)
		}
	}

	return nil
}