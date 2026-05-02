package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/ports"
	"simple-orderbook/internal/db"
)

type OrderWorker struct {
	consumer     ports.KafkaReader
	orderBook    *domain.OrderBook
	cache        ports.OrderBookCache
	orderRepo    ports.OrderRepository
	tradeRepo    ports.TradeRepository
	metrics      ports.Metrics
	broadcaster  ports.Broadcaster
	tradesBuf    []domain.Trade
	store        *db.Store
	mu           sync.RWMutex
	needsRefresh atomic.Bool
}

func NewOrderWorker(consumer ports.KafkaReader, orderBook *domain.OrderBook, cache ports.OrderBookCache, orderRepo ports.OrderRepository, tradeRepo ports.TradeRepository, metrics ports.Metrics, broadcaster ports.Broadcaster, store *db.Store) *OrderWorker {
	return &OrderWorker{
		consumer:    consumer,
		orderBook:   orderBook,
		cache:       cache,
		orderRepo:   orderRepo,
		tradeRepo:   tradeRepo,
		metrics:     metrics,
		broadcaster: broadcaster,
		tradesBuf:   make([]domain.Trade, 1024),
		store:       store,
	}
}

func (w *OrderWorker) snapshotLoop(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if w.needsRefresh.CompareAndSwap(true, false) {
				w.broadcastAndCache(ctx)
			}
		}
	}
}

func (w *OrderWorker) broadcastAndCache(ctx context.Context) {
	w.mu.RLock()
	snap := w.orderBook.Snapshot()
	w.mu.RUnlock()

	payload, err := json.Marshal(snap)
	if err != nil {
		slog.Error("worker: snapshot marshal failed", "error", err)
		return
	}

	_ = w.broadcaster.Broadcast(ctx, ports.BroadcastEvent{
		Type:    "ORDERBOOK_UPDATE",
		Payload: payload,
	})

	if err := w.cache.Set(ctx, payload); err != nil {
		slog.Error("worker: cache update failed", "error", err)
	}
}

func (w *OrderWorker) handlePlaceOrder(ctx context.Context, payload []byte) {
	var order domain.Order
	if err := json.Unmarshal(payload, &order); err != nil {
		slog.Error("worker: unmarshal error", "error", err)
		return
	}

	engineStart := time.Now()
	w.mu.Lock()
	trades, takerStatus := w.orderBook.PlaceOrder(order.ID, order.UserID, order.Price, order.Quantity, order.Side, w.tradesBuf)
	w.mu.Unlock()

	w.metrics.RecordMatchingLatency(time.Since(engineStart))

	// Restore tradesBuf
	w.tradesBuf = trades[:0]

	err := w.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := w.orderRepo.UpdateStatus(ctx, q, order.ID, takerStatus); err != nil {
			return err
		}
		if len(trades) > 0 {
			if err := w.tradeRepo.CreateBatch(ctx, q, trades, order.Side); err != nil {
				return err
			}

			makerIDs := make([]uuid.UUID, len(trades))
			makerStatuses := make([]domain.OrderStatus, len(trades))

			for i, trade := range trades {
				makerIDs[i] = trade.MakerOrderID

				w.mu.RLock()
				m, ok := w.orderBook.GetOrder(trade.MakerOrderID)

				makerStatuses[i] = domain.StatusFilled
				if ok {
					makerStatuses[i] = m.Status
				}
				w.mu.RUnlock()

				slog.Info("Worker matched trade", "takerID", order.ID, "makerID", trade.MakerOrderID, "takerStatus", takerStatus)
			}

			return w.orderRepo.UpdateOrderStatusBatch(ctx, q, makerIDs, makerStatuses)
		}

		return nil
	})

	if err != nil {
		slog.Error("worker: DB transaction failed", "error", err, "order_id", order.ID)
		return
	}

	if len(trades) > 0 {
		go func(t []domain.Trade) {
			tradePayload, _ := json.Marshal(t)
			_ = w.broadcaster.Broadcast(context.Background(), ports.BroadcastEvent{
				Type:    "TRADES_EXECUTED",
				Payload: tradePayload,
			})
		}(trades)

		w.metrics.RecordTrade(int64(len(trades)))
	}

	w.needsRefresh.Store(true)

	w.metrics.RecordEndToEndLatency(time.Since(time.Unix(0, order.CreatedAt)))
}

func (w *OrderWorker) handleCancelOrder(ctx context.Context, payload json.RawMessage) {
	var req struct {
		OrderID uuid.UUID `json:"order_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		slog.Error("worker: cancel unmarshal error", "error", err)
		return
	}

	w.mu.Lock()
	ok := w.orderBook.CancelOrder(req.OrderID)
	w.mu.Unlock()

	if ok {
		err := w.store.ExecTx(ctx, func(q *db.Queries) error {
			return w.orderRepo.UpdateStatus(ctx, q, req.OrderID, domain.StatusCancelled)
		})

		if err != nil {
			slog.Error("worker: cancel DB update failed", "error", err)
			return
		}

		w.needsRefresh.Store(true)

		slog.Info("worker: order cancelled", "order_id", req.OrderID)
	}
}

//nolint:gocritic // msg is passed by value from kafka-go
func (w *OrderWorker) handleMessages(ctx context.Context, msg kafka.Message) {
	var eventType string
	for _, h := range msg.Headers {
		if h.Key == "event_type" {
			eventType = string(h.Value)
			break
		}
	}

	switch eventType {
	case "OrderPlaced":
		w.handlePlaceOrder(ctx, msg.Value)
	case "OrderCancelRequested":
		w.handleCancelOrder(ctx, msg.Value)
	default:
		slog.Warn("worker: unknown command type", "type", eventType)
	}
}

func (w *OrderWorker) Run(ctx context.Context) {
	slog.Info("order worker started")

	go w.snapshotLoop(ctx)

	for {
		msg, err := w.consumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("worker: read error", "error", err)
			continue
		}

		w.handleMessages(ctx, msg)

		if err := w.consumer.CommitMessage(ctx, msg); err != nil {
			slog.Error("worker: commit error", "error", err)
		}
	}
}
