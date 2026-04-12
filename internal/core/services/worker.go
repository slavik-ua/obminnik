package services

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/ports"
	"simple-orderbook/internal/db"
)

type OrderWorker struct {
	consumer  ports.KafkaReader
	orderBook *domain.OrderBook
	cache     ports.OrderBookCache
	orderRepo ports.OrderRepository
	tradeRepo ports.TradeRepository
	store     *db.Store
}

func NewOrderWorker(consumer ports.KafkaReader, orderBook *domain.OrderBook, cache ports.OrderBookCache, orderRepo ports.OrderRepository, tradeRepo ports.TradeRepository, store *db.Store) *OrderWorker {
	return &OrderWorker{
		consumer:  consumer,
		orderBook: orderBook,
		cache:     cache,
		orderRepo: orderRepo,
		tradeRepo: tradeRepo,
		store:     store,
	}
}

func (w *OrderWorker) refreshCache(ctx context.Context) {
	if err := w.cache.Invalidate(ctx); err != nil {
		slog.Error("worker: cache invalidation failed", "error", err)
	}
}

func (w *OrderWorker) handlePlaceOrder(ctx context.Context, payload []byte) {
	var order domain.Order
	if err := json.Unmarshal(payload, &order); err != nil {
		slog.Error("worker: unmarshal error", "error", err)
		return
	}

	trades := w.orderBook.PlaceOrder(order.ID, order.UserID, order.Price, order.Quantity, order.Side, nil)

	engineOrder, _ := w.orderBook.GetOrder(order.ID)
	status := engineOrder.Status

	err := w.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := w.orderRepo.UpdateStatus(ctx, q, order.ID, status); err != nil {
			return err
		}
		for _, trade := range trades {
			if err := w.tradeRepo.Create(ctx, q, &trade); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		slog.Error("worker: DB transaction failed", "error", err, "order_id", order.ID)
		return
	}

	w.refreshCache(ctx)

	slog.Info("worker: processed order", "order_id", order.ID, "trades", len(trades), "status", status)
}

func (w *OrderWorker) handleCancelOrder(ctx context.Context, payload json.RawMessage) {
	var req struct {
		OrderID uuid.UUID `json:"order_id"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		slog.Error("worker: cancel unmarshal error", "error", err)
		return
	}

	if ok := w.orderBook.CancelOrder(req.OrderID); ok {
		err := w.store.ExecTx(ctx, func(q *db.Queries) error {
			return w.orderRepo.UpdateStatus(ctx, q, req.OrderID, domain.StatusCancelled)
		})

		if err != nil {
			slog.Error("worker: cancel DB update failed", "error", err)
			return
		}

		w.refreshCache(ctx)
		slog.Info("worker: order cancelled", "order_id", req.OrderID)
	}
}

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
