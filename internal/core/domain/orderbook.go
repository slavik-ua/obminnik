package domain

import (
	"sync"
	"slices"
	"cmp"

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
	mu        *sync.RWMutex
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids:   make(map[int64]*PriceLevel),
		Asks:   make(map[int64]*PriceLevel),
		Orders: make(map[uuid.UUID]*Order),
		mu:     &sync.RWMutex{},
	}
}

func (ob *OrderBook) AddOrder(id uuid.UUID, price int64, quantity int64, side OrderSide) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order := &Order{ID: id, Price: price, Quantity: quantity, RemainingQuantity: quantity, Side: side, Status: StatusNew}

	var levels map[int64]*PriceLevel
	if side == SideBuy {
		levels = ob.Bids
	} else {
		levels = ob.Asks
	}

	if _, ok := levels[price]; !ok {
		levels[price] = &PriceLevel{Price: price}

		if side == SideBuy {
			idx, _ := slices.BinarySearchFunc(ob.BidsIndex, price, func(e, target int64) int {
				return cmp.Compare(target, e)
			})
			ob.BidsIndex = slices.Insert(ob.BidsIndex, idx, price)
		} else {
			idx, _ := slices.BinarySearchFunc(ob.AsksIndex, price, func(e, target int64) int {
				return cmp.Compare(e, target)
			})
			ob.AsksIndex = slices.Insert(ob.AsksIndex, idx, price)
		}
	}

	level := levels[price]
	if level.Tail == nil {
		level.Head = order
		level.Tail = order
	} else {
		order.prev = level.Tail
		level.Tail.next = order
		level.Tail = order
	}
	level.TotalVol += quantity
	order.parent = level
	ob.Orders[id] = order
}	

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

	level.TotalVol -= order.RemainingQuantity
	delete(ob.Orders, order.ID)

	if level.Head == nil {
		if order.Side == SideBuy {
			ob.BidsIndex = slices.DeleteFunc(ob.BidsIndex, func(p int64) bool { return p == order.Price })
			delete(ob.Bids, order.Price)
		} else {
			ob.AsksIndex = slices.DeleteFunc(ob.AsksIndex, func(p int64) bool { return p == order.Price })
			delete(ob.Asks, order.Price)
		}
	}

	order.next = nil
	order.prev = nil
}

func (ob *OrderBook) CancelOrder(id uuid.UUID) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, ok := ob.Orders[id]
	if !ok {
		return
	}

	order.Status = StatusCancelled
	ob.removeOrderInternal(order)
}

func min(a, b int64) int64 {
	if a < b { return a }
	return b
}

func (ob *OrderBook) Match(takeOrder *Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	for takeOrder.RemainingQuantity > 0 {
		var levelPrice int64
		var levels map[int64]*PriceLevel

		if takeOrder.Side == SideBuy {
			if len(ob.AsksIndex) == 0 { break }
			levelPrice = ob.AsksIndex[0]

			if takeOrder.Price < levelPrice { break }
			levels = ob.Asks
		} else {
			if len(ob.BidsIndex) == 0 { break }
			levelPrice = ob.BidsIndex[0]

			if takeOrder.Price > levelPrice { break }
			levels = ob.Bids
		}

		level := levels[levelPrice]
		currentMaker := level.Head

		for currentMaker != nil && takeOrder.RemainingQuantity > 0 {
			tradeQty := min(takeOrder.RemainingQuantity, currentMaker.RemainingQuantity)

			takeOrder.RemainingQuantity -= tradeQty
			currentMaker.RemainingQuantity -= tradeQty

			if currentMaker.RemainingQuantity == 0 {
				currentMaker.Status = StatusFilled
				nextOrder := currentMaker.next
				ob.removeOrderInternal(currentMaker)
				currentMaker = nextOrder
			}
		}
	}

	if takeOrder.RemainingQuantity == 0 {
		takeOrder.Status = StatusFilled
	}
}