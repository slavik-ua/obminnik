package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

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

	idGen domain.IDGenerator
}

func NewOrderService(
	store *db.Store,
	orderRepo ports.OrderRepository,
	outboxRepo ports.OutboxRepository,
	book *domain.OrderBook,
	cache ports.OrderBookCache,
	notifier ports.OutboxNotifier,
	idGen domain.IDGenerator,
) *OrderService {
	return &OrderService{
		store:      store,
		orderRepo:  orderRepo,
		outboxRepo: outboxRepo,
		book:       book,
		cache:      cache,
		notifier:   notifier,
		idGen:      idGen,
	}
}

func (s *OrderService) PlaceOrder(ctx context.Context, order *domain.Order) error {
	if order.ID == uuid.Nil {
		order.ID = s.idGen.Next()
	}
	if order.CreatedAt == 0 {
		order.CreatedAt = time.Now().UnixNano()
	}

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
			ID:      s.idGen.Next(),
			Type:    "OrderPlaced",
			Payload: payload,
		}
		return s.outboxRepo.AddEvent(ctx, q, event)
	})

	if err == nil {
		s.notifier.Notify()
	}

	return err
}

func (s *OrderService) CancelOrder(ctx context.Context, id uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		payload, err := json.Marshal(map[string]interface{}{"order_id": id})
		if err != nil {
			return err
		}

		event := &domain.OutboxEvent{
			ID:      s.idGen.Next(),
			Type:    "OrderCancelRequested",
			Payload: payload,
		}
		return s.outboxRepo.AddEvent(ctx, q, event)
	})
}

func (s *OrderService) GetOrderBook(ctx context.Context) ([]byte, error) {
	data, found, err := s.cache.Get(ctx)
	if err != nil {
		// fail-open
		slog.Warn("orderService: orderbook cache get failed", "error", err)
		return nil, err
	}
	if found {
		return data, nil
	}
	slog.Info("orderService: cache miss")
	return nil, err
}

func (s *OrderService) RebuildOrderBook(ctx context.Context) error {
	slog.Info("orderService: rebuilding orderbook from the database")
	for _, side := range []db.OrderSide{db.OrderSideBUY, db.OrderSideSELL} {
		orders, err := s.orderRepo.ListActiveBySide(ctx, side)
		if err != nil {
			return fmt.Errorf("rebuild orderbook: %w", err)
		}
		for _, o := range orders {
			s.book.RestoreOrder(o)
		}
	}
	slog.Info("orderService: orderbook rebuild complete")

	snapshot := s.book.Snapshot()
	payload, err := json.Marshal(snapshot)
	if err != nil {
		slog.Error("orderService: failed to marshal OrderBook snapshot")
		return err
	}
	if err := s.cache.Set(ctx, payload); err != nil {
		return err
	}

	return nil
}
