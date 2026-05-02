package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/ports"
	"simple-orderbook/internal/db"
)

type OrderService struct {
	orderRepo   ports.OrderRepository
	accountRepo ports.AccountRepository
	outboxRepo  ports.OutboxRepository
	store       *db.Store

	book     *domain.OrderBook
	cache    ports.OrderBookCache
	notifier ports.OutboxNotifier

	idGen domain.IDGenerator
}

func NewOrderService(
	store *db.Store,
	orderRepo ports.OrderRepository,
	accountRepo ports.AccountRepository,
	outboxRepo ports.OutboxRepository,
	book *domain.OrderBook,
	cache ports.OrderBookCache,
	notifier ports.OutboxNotifier,
	idGen domain.IDGenerator,
) *OrderService {
	return &OrderService{
		store:       store,
		orderRepo:   orderRepo,
		accountRepo: accountRepo,
		outboxRepo:  outboxRepo,
		book:        book,
		cache:       cache,
		notifier:    notifier,
		idGen:       idGen,
	}
}

func (s *OrderService) CancelOrder(ctx context.Context, userID, orderID uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q *db.Queries) error {
		// Check if order exists and belongs to user
		order, err := s.orderRepo.GetByID(ctx, orderID)
		if err != nil {
			return err
		}
		if order.UserID != userID {
			return errors.New("unauthorized: order does not belong to user")
		}
		if order.Status != domain.StatusPlaced && order.Status != domain.StatusPartial && order.Status != domain.StatusNew {
			return errors.New("cannot cancel order in current status")
		}

		payload, _ := json.Marshal(map[string]interface{}{
			"order_id": orderID,
		})

		event := &domain.OutboxEvent{
			ID:      s.idGen.Next(),
			Type:    "OrderCancelRequested",
			Payload: payload,
		}

		if err := s.outboxRepo.AddEvent(ctx, q, event); err != nil {
			return err
		}

		s.notifier.Notify()
		return nil
	})
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

func (s *OrderService) Deposit(ctx context.Context, userID uuid.UUID, asset string, amount int64) error {
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		err := s.accountRepo.Deposit(ctx, q, userID, asset, amount)
		if err != nil {
			return err
		}

		payload, _ := json.Marshal(map[string]interface{}{
			"user_id": userID,
			"asset":   asset,
			"amount":  amount,
		})

		event := &domain.OutboxEvent{
			ID:      s.idGen.Next(),
			Type:    "DepositCreated",
			Payload: payload,
		}

		return s.outboxRepo.AddEvent(ctx, q, event)
	})

	if err == nil {
		s.notifier.Notify()
	}
	return err
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
func (s *OrderService) GetBalances(ctx context.Context, userID uuid.UUID) ([]domain.BalanceRecord, error) {
	return s.accountRepo.GetBalances(ctx, userID)
}
