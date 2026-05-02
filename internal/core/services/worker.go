package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/big"
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
	accountRepo  ports.AccountRepository
	balanceCache *domain.BalanceCache
	metrics      ports.Metrics
	broadcaster  ports.Broadcaster
	tradesBuf    []domain.Trade
	store        *db.Store
	mu           sync.RWMutex
	needsRefresh atomic.Bool
}

func NewOrderWorker(consumer ports.KafkaReader, orderBook *domain.OrderBook, cache ports.OrderBookCache, orderRepo ports.OrderRepository, tradeRepo ports.TradeRepository, accountRepo ports.AccountRepository, balanceCache *domain.BalanceCache, metrics ports.Metrics, broadcaster ports.Broadcaster, store *db.Store) *OrderWorker {
	return &OrderWorker{
		consumer:     consumer,
		orderBook:    orderBook,
		cache:        cache,
		orderRepo:    orderRepo,
		tradeRepo:    tradeRepo,
		accountRepo:  accountRepo,
		balanceCache: balanceCache,
		metrics:      metrics,
		broadcaster:  broadcaster,
		tradesBuf:    make([]domain.Trade, 1024),
		store:        store,
	}
}

type balanceDelta struct {
	available int64
	locked    int64
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

func (w *OrderWorker) persistTradeSettlements(ctx context.Context, q *db.Queries, trades []domain.Trade, takerOrder *domain.Order) error {
	deltas := make(map[string]map[string]*balanceDelta)

	addDelta := func(userID uuid.UUID, asset string, avail, lock int64) {
		uStr := userID.String()
		if _, ok := deltas[uStr]; !ok {
			deltas[uStr] = make(map[string]*balanceDelta)
		}
		if _, ok := deltas[uStr][asset]; !ok {
			deltas[uStr][asset] = &balanceDelta{}
		}
		deltas[uStr][asset].available += avail
		deltas[uStr][asset].locked += lock
	}

	// Calculate deltas based on trades
	baseAsset := "BTC"
	quoteAsset := "USD"

	for _, t := range trades {
		p := big.NewInt(t.Price)
		q := big.NewInt(t.Quantity)
		d := big.NewInt(domain.Decimals)
		amtBig := new(big.Int).Mul(p, q)
		amtBig.Quo(amtBig, d)
		quoteAmount := amtBig.Int64()

		if takerOrder.Side == domain.SideBuy {
			// Taker is Buyer: Pays Quote (USD), Receives Base (BTC)
			addDelta(t.TakerUserID, quoteAsset, 0, -quoteAmount)
			addDelta(t.TakerUserID, baseAsset, t.Quantity, 0)

			// Maker is Seller: Pays Base (BTC), Receives Quote (USD)
			addDelta(t.MakerUserID, baseAsset, 0, -t.Quantity)
			addDelta(t.MakerUserID, quoteAsset, quoteAmount, 0)
		} else {
			// Taker is Seller: Pays Base (BTC), Receives Base (USD)
			addDelta(t.TakerUserID, baseAsset, 0, -t.Quantity)
			addDelta(t.TakerUserID, quoteAsset, quoteAmount, 0)

			// Maker is Buyer: Pays Base (USD), Receives Quote (BTC)
			addDelta(t.MakerUserID, quoteAsset, 0, -quoteAmount)
			addDelta(t.MakerUserID, baseAsset, t.Quantity, 0)
		}
	}

	var userIDs []uuid.UUID
	var assets []string
	var availDeltas []int64
	var lockDeltas []int64

	for uIDStr, assetMap := range deltas {
		uID := uuid.MustParse(uIDStr)
		for asset, d := range assetMap {
			userIDs = append(userIDs, uID)
			assets = append(assets, asset)
			availDeltas = append(availDeltas, d.available)
			lockDeltas = append(lockDeltas, d.locked)
		}
	}

	if err := q.EnsureBalancesExist(ctx, db.EnsureBalancesExistParams{
		UserIds: userIDs,
		Assets:  assets,
	}); err != nil {
		return err
	}

	return q.BatchUpdateBalances(ctx, db.BatchUpdateBalancesParams{
		UserIds:     userIDs,
		Assets:      assets,
		AvailDeltas: availDeltas,
		LockDeltas:  lockDeltas,
	})
}

func (w *OrderWorker) handlePlaceOrder(ctx context.Context, payload []byte) {
	var order domain.Order
	if err := json.Unmarshal(payload, &order); err != nil {
		slog.Error("worker: unmarshal error", "error", err)
		return
	}

	var lockAsset string
	var lockAmount int64
	if order.Side == domain.SideBuy {
		lockAsset = "USD"
		// Use big.Int to prevent overflow: (Price * Quantity) / Decimals
		p := big.NewInt(order.Price)
		q := big.NewInt(order.Quantity)
		d := big.NewInt(domain.Decimals)
		amt := new(big.Int).Mul(p, q)
		amt.Quo(amt, d)
		lockAmount = amt.Int64()
	} else {
		lockAsset = "BTC"
		lockAmount = order.Quantity
	}

	slog.Info("Worker locking funds", "user", order.UserID, "asset", lockAsset, "amount", lockAmount)
	w.mu.Lock()
	err := w.balanceCache.LockFunds(order.UserID, lockAsset, lockAmount)
	w.mu.Unlock()

	if err != nil {
		slog.Error("worker: memory lock failed", "err", err, "user", order.UserID, "asset", lockAsset, "amount", lockAmount)
		return
	}

	engineStart := time.Now()
	w.mu.Lock()
	trades, takerStatus := w.orderBook.PlaceOrder(order.ID, order.UserID, order.Price, order.Quantity, order.Side, w.tradesBuf)

	if len(trades) > 0 {
		for _, trade := range trades {
			var buyer, seller uuid.UUID
			if order.Side == domain.SideBuy {
				buyer = trade.TakerUserID
				seller = trade.MakerUserID
			} else {
				buyer = trade.MakerUserID
				seller = trade.TakerUserID
			}

			errSettle := w.balanceCache.SettleTrade(
				buyer,
				seller,
				"BTC", "USD",
				trade.Price,
				trade.Quantity,
			)
			if errSettle != nil {
				slog.Error("worker: settlement failed", "trade_id", trade.ID, "err", errSettle)
				panic("memory is inconsistent")
			}
		}
	}

	w.mu.Unlock()

	w.metrics.RecordMatchingLatency(time.Since(engineStart))

	// Restore tradesBuf
	w.tradesBuf = trades[:0]

	err = w.store.ExecTx(ctx, func(q *db.Queries) error {
		_, errLock := q.UpdateBalanceLock(ctx, db.UpdateBalanceLockParams{
			UserID:      order.UserID,
			AssetSymbol: lockAsset,
			Amount:      lockAmount,
		})
		if errLock != nil {
			return errLock
		}

		if errStatus := w.orderRepo.UpdateStatus(ctx, q, order.ID, takerStatus); errStatus != nil {
			return errStatus
		}

		if len(trades) > 0 {
			if errBatch := w.tradeRepo.CreateBatch(ctx, q, trades, order.Side); errBatch != nil {
				return errBatch
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
			}

			if errBatch := w.orderRepo.UpdateOrderStatusBatch(ctx, q, makerIDs, makerStatuses); errBatch != nil {
				return errBatch
			}

			if errSettlement := w.persistTradeSettlements(ctx, q, trades, &order); errSettlement != nil {
				return errSettlement
			}
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

	w.mu.RLock()
	order, ok := w.orderBook.GetOrder(req.OrderID)
	w.mu.RUnlock()

	if !ok {
		slog.Warn("worker: cancel requested for non-existent order", "id", req.OrderID)
		return
	}

	w.mu.Lock()
	cancelled := w.orderBook.CancelOrder(req.OrderID)
	w.mu.Unlock()

	if !cancelled {
		return
	}

	var unlockAsset string
	var unlockAmount int64
	if order.Side == domain.SideBuy {
		unlockAsset = "USD"
		p := big.NewInt(order.Price)
		q := big.NewInt(order.RemainingQuantity)
		d := big.NewInt(domain.Decimals)
		amt := new(big.Int).Mul(p, q)
		amt.Quo(amt, d)
		unlockAmount = amt.Int64()
	} else {
		unlockAsset = "BTC"
		unlockAmount = order.RemainingQuantity
	}

	w.mu.Lock()
	err := w.balanceCache.UnlockFunds(order.UserID, unlockAsset, unlockAmount)
	w.mu.Unlock()

	if err != nil {
		slog.Error("worker: memory lock failed (out of sync)", "err", err, "user", order.UserID)
		return
	}

	err = w.store.ExecTx(ctx, func(q *db.Queries) error {
		_, errUnlock := q.UpdateBalanceUnlock(ctx, db.UpdateBalanceUnlockParams{
			UserID:      order.UserID,
			AssetSymbol: unlockAsset,
			Amount:      unlockAmount,
		})
		if errUnlock != nil {
			return errUnlock
		}
		return w.orderRepo.UpdateStatus(ctx, q, req.OrderID, domain.StatusCancelled)
	})

	if err != nil {
		slog.Error("worker: CRITICAL cancel DB update failed", "error", err)
		panic("database out of sync with matching engine")
	}

	w.needsRefresh.Store(true)

	slog.Info("worker: order cancelled and funds unlocked", "order_id", req.OrderID, "user", order.UserID, "amount", unlockAmount)
}

//nolint:gocritic // msg is passed by value from kafka-go
func (w *OrderWorker) handleMessages(ctx context.Context, msg kafka.Message) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("WORKER CRITICAL PANIC", "recover", r)
		}
	}()

	var eventType string
	for _, h := range msg.Headers {
		if h.Key == "event_type" {
			eventType = string(h.Value)
			break
		}
	}

	if eventType == "" {
		return
	}

	slog.Info("Worker processing event", "type", eventType)

	switch eventType {
	case "OrderPlaced":
		w.handlePlaceOrder(ctx, msg.Value)
	case "OrderCancelRequested":
		w.handleCancelOrder(ctx, msg.Value)
	case "DepositCreated":
		var dep struct {
			UserID uuid.UUID `json:"user_id"`
			Asset  string    `json:"asset"`
			Amount int64     `json:"amount"`
		}
		if err := json.Unmarshal(msg.Value, &dep); err != nil {
			slog.Error("worker: deposit unmarshal error", "error", err)
			return
		}

		w.mu.Lock()
		w.balanceCache.Deposit(dep.UserID, dep.Asset, dep.Amount)
		w.mu.Unlock()

		slog.Info("worker: balance updated in memory", "user", dep.UserID, "asset", dep.Asset, "amount", dep.Amount)
	default:
		slog.Warn("worker: unknown command type", "type", eventType)
	}
}

func (w *OrderWorker) hydrate(ctx context.Context) error {
	slog.Info("worker: hydrating balance cache from database")

	records, err := w.accountRepo.ListAllBalances(ctx)
	if err != nil {
		return err
	}

	for _, r := range records {
		w.balanceCache.InitBalance(r.UserID, r.AssetSymbol, r.Available, r.Locked)
	}

	slog.Info("worker: hydration complete", "count", len(records))
	return nil
}

func (w *OrderWorker) Run(ctx context.Context) {
	if err := w.hydrate(ctx); err != nil {
		slog.Error("worker: fatal hydration error", "error", err)
		return
	}

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

// Reset cleanly resets the internal matching engine state (Book + Balances)
func (w *OrderWorker) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.orderBook.Reset()
	w.balanceCache.Clear()
}
