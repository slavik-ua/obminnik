package domain

import (
	"sync"
	"slices"
	"cmp"
	"time"

	"github.com/google/uuid"
)

type PriceLevel struct {
	Price    int64
	TotalVol int64
	Head     *Order
	Tail     *Order
}

type OrderBook struct {
	Bids      map[int64]*PriceLevel
	Asks      map[int64]*PriceLevel
	BidsIndex []int64
	AsksIndex []int64
	Orders    map[uuid.UUID]*Order
	mu        sync.RWMutex
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids:   make(map[int64]*PriceLevel),
		Asks:   make(map[int64]*PriceLevel),
		Orders: make(map[uuid.UUID]*Order),
	}
}

// PlaceOrder is the single entry point for submitting a limit order
// It matches against the opposite side first, then rests any unfilled
// remainder on the book
// 
// trades is a caller-owned buffer used to collect fills. Pass a non-nil
// slice to reuse its backing array across calls and avoid per-call heap
// allocations. The slice is reset to length 0 on entry. The returned slice
// shares the same backing array
func (ob *OrderBook) PlaceOrder(id uuid.UUID, userID uuid.UUID, price, quantity int64, side OrderSide, trades []Trade) []Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order := &Order{
		ID:                id,
		UserID:            userID,
		CreatedAt:         time.Now().UnixNano(),
		Price:             price,
		Quantity:          quantity,
		RemainingQuantity: quantity,
		Side:              side,
		Status:            StatusNew,
	}

	trades = ob.matchInternal(order, trades[:0])

	switch {
	case order.RemainingQuantity == 0:
		order.Status = StatusFilled
	case order.RemainingQuantity < order.Quantity:
		order.Status = StatusPartial
		ob.addOrderInternal(order)
	default:
		ob.addOrderInternal(order)
	}

	return trades
}

func (ob *OrderBook) CancelOrder(id uuid.UUID) bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, ok := ob.Orders[id]
	if !ok {
		return false
	}

	order.parent.TotalVol -= order.RemainingQuantity
	order.Status = StatusCancelled
	ob.removeOrderInternal(order)
	return true
}

func (ob *OrderBook) GetOrder(id uuid.UUID) (*Order, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	o, ok := ob.Orders[id]
	return o, ok
}

// matchInternal walks the opposite side of the book and fills as much of
// taker as possible. TotalVol is decremented per trade, not per removal,
// so it always reflects actual available volume
func (ob *OrderBook) matchInternal(taker *Order, trades []Trade) []Trade {
	for taker.RemainingQuantity > 0 {
		levelPrice, levels, ok := ob.bestOposite(taker)
		if !ok {
			break
		}

		level := levels[levelPrice]
		maker := level.Head

		for maker != nil && taker.RemainingQuantity > 0 {
			tradeQty := min(taker.RemainingQuantity, maker.RemainingQuantity)

			taker.RemainingQuantity -= tradeQty
			maker.RemainingQuantity -= tradeQty

			level.TotalVol -= tradeQty

			trades = append(trades, Trade{
				Price:        levelPrice,
				Quantity:     tradeQty,
				TakerOrderID: taker.ID,
				MakerOrderID: maker.ID,
				TakerUserID:  taker.UserID,
				MakerUserID:  maker.UserID,
			})

			next := maker.next
			if maker.RemainingQuantity == 0 {
				maker.Status = StatusFilled
				ob.removeOrderInternal(maker)
			}
			maker = next
		}
	}

	return trades
}

func (ob *OrderBook) bestOposite(taker *Order) (int64, map[int64]*PriceLevel, bool) {
	if taker.Side == SideBuy {
		if len(ob.AsksIndex) == 0 {
			return 0, nil, false
		}
		best := ob.AsksIndex[0]
		if taker.Price < best {
			return 0, nil, false
		}

		return best, ob.Asks, true
	}

	if len(ob.BidsIndex) == 0 {
		return 0, nil, false
	}
	best := ob.BidsIndex[0]
	if taker.Price > best {
		return 0, nil, false
	}
	return best, ob.Bids, true
}

// addOrderInternal appends order to the correct price level,
// creating the level (and updating the sorted index) if necessary
func (ob *OrderBook) addOrderInternal(order *Order) {
	levels, index, sortCmp := ob.sideData(order.Side)

	level, exists := levels[order.Price]
	if !exists {
		level = &PriceLevel{Price: order.Price}
		levels[order.Price] = level
		idx, _ := slices.BinarySearchFunc(*index, order.Price, sortCmp)
		*index = slices.Insert(*index, idx, order.Price)
	}

	if level.Tail == nil {
		level.Head = order
		level.Tail = order
	} else {
		order.prev = level.Tail
		level.Tail.next = order
		level.Tail = order
	}

	level.TotalVol += order.RemainingQuantity
	order.parent = level
	ob.Orders[order.ID] = order
}

// removeOrderInternal unlinks order from its price level and removes it from
// the Orders map. It does not touch TotalVol - callers are responsible for
// adjusting volume before calling this function
func (ob *OrderBook) removeOrderInternal(order *Order) {
	level := order.parent

	if order.prev != nil {
		order.prev.next = order.next
	} else {
		level.Head = order.next
	}

	if order.next != nil {
		order.next.prev = order.prev
	} else {
		level.Tail = order.prev
	}

	delete(ob.Orders, order.ID)

	if level.Head == nil {
		levels, index, sortCmp := ob.sideData(order.Side)
		if idx, found := slices.BinarySearchFunc(*index, order.Price, sortCmp); found {
			*index = slices.Delete(*index, idx, idx+1)
		}
		delete(levels, order.Price)
	}

	// Clear pointers to prevent dangling references
	order.next = nil
	order.prev = nil
	order.parent = nil
}

// Returns the map, sorted index slice pointer, and comparison function
func (ob *OrderBook) sideData(side OrderSide) (map[int64]*PriceLevel, *[]int64, func(int64, int64) int) {
	if side == SideBuy {
		return ob.Bids, &ob.BidsIndex, func(e, t int64) int { return cmp.Compare(t, e) }
	}
	return ob.Asks, &ob.AsksIndex, func(e, t int64) int { return cmp.Compare(e, t) }
}

func min(a, b int64) int64 {
	if (a < b) {
		return a
	}
	return b
}