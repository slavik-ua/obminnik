package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/ports"
	"simple-orderbook/internal/db"
)

type OrderService struct {
	orderRepo  ports.OrderRepository
	outboxRepo ports.OutboxRepository
	store      *db.Store

	book     *domain.OrderBook
	cache    ports.OrderBookCache
	notifier ports.OutboxNotifier
}

func NewOrderService(
	store *db.Store,
	orderRepo ports.OrderRepository,
	outboxRepo ports.OutboxRepository,
	book *domain.OrderBook,
	cache ports.OrderBookCache,
	notifier ports.OutboxNotifier,
) *OrderService {
	return &OrderService{
		store:      store,
		orderRepo:  orderRepo,
		outboxRepo: outboxRepo,
		book:       book,
		cache:      cache,
		notifier:   notifier,
	}
}

func (s *OrderService) PlaceOrder(ctx context.Context, order *domain.Order) error {
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		order.Status = domain.StatusNew
		if err := s.orderRepo.Create(ctx, q, order); err != nil {
			return err
		}

		payload, err := json.Marshal(order)
		if err != nil {
			return err
		}

		event := &domain.OutboxEvent{
			ID:      uuid.New(),
			Type:    "OrderPlaced",
			Payload: payload,
		}
		return s.outboxRepo.AddEvent(ctx, q, event)
	})

	if err == nil {
		s.notifier.Notify()
	}

	return err
	/*
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
			slog.Warn("cache invalidation failed after order placement",
				"error", err,
				"order_id", order.ID,
			)
		}

		return trades, err
	*/
}

func (s *OrderService) CancelOrder(ctx context.Context, id uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		payload, err := json.Marshal(map[string]interface{}{"order_id": id})
		if err != nil {
			return err
		}

		event := &domain.OutboxEvent{
			ID:      uuid.New(),
			Type:    "OrderCancelRequested",
			Payload: payload,
		}
		return s.outboxRepo.AddEvent(ctx, q, event)
	})

	/*
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
			slog.Warn("cache invalidation failed after cancel",
				"error", err,
				"order_id", id,
			)
		}

		return nil
	*/
}

func (s *OrderService) GetOrderBook(ctx context.Context) ([]byte, error) {
	data, found, err := s.cache.Get(ctx)
	if err != nil {
		// fail-open
		slog.Warn("ordebook cache get failed", "error", err)
	}
	if found {
		return data, nil
	}
	slog.Info("cache miss, falling back to in-memory snapshot")

	snapshot := s.book.Snapshot()
	data, err = json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal orderbook: %w", err)
	}

	if err := s.cache.Set(ctx, data); err != nil {
		slog.Warn("failed to populate cache", "error", err)
	}

	return data, nil
}

func (s *OrderService) RebuildOrderBook(ctx context.Context) error {
	slog.Info("rebuilding order book from database")
	for _, side := range []db.OrderSide{db.OrderSideBUY, db.OrderSideSELL} {
		orders, err := s.orderRepo.ListActiveBySide(ctx, side)
		if err != nil {
			return fmt.Errorf("rebuild orderbook: %w", err)
		}
		for _, o := range orders {
			s.book.RestoreOrder(o)
		}
	}
	slog.Info("order book rebuild complete")
	return nil
}
