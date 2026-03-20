package domain

import (
	"sync"
)

type PriceLevel struct {
	Price    int
	TotalVol int
	Head     *Order
	Tail     *Order
}

type OrderBook struct {
	Bids   map[int]*PriceLevel
	Asks   map[int]*PriceLevel
	Orders map[string]*Order
	mu     *sync.RWMutex
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids:   make(map[int]*PriceLevel),
		Asks:   make(map[int]*PriceLevel),
		Orders: make(map[string]*Order),
		mu:     &sync.RWMutex{},
	}
}

func (ob *OrderBook) AddOrder(id string, price int, quantity int, side OrderSide) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order := &Order{ID: id, Price: price, Quantity: quantity, RemainingQuantity: quantity, Side: side, Status: StatusNew}

	var levels map[int]*PriceLevel
	if side == SideBuy {
		levels = ob.Bids
	} else {
		levels = ob.Asks
	}

	if _, ok := levels[price]; !ok {
		levels[price] = &PriceLevel{Price: price}
	}
	level := levels[price]

	if level.Tail == nil {
		level.Head = order
		level.Tail = order
	} else {
		order.Prev = level.Tail
		level.Tail.Next = order
		level.Tail = order
	}
	level.TotalVol += quantity

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
		level.Tail = order.Prev
	}

	level.TotalVol -= int(order.RemainingQuantity)
	delete(ob.Orders, order.ID.String())

	order.next = nil
	order.prev = nil
}

func (ob *OrderBook) CancelOrder(id string) {
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

	var level *PriceLevel
	if takeOrder.Side == SideBuy {
		level = ob.Asks[int(takeOrder.Price)]
	} else {
		level = ob.Bids[int(takeOrder.Price)]
	}

	if level == nil || level.Head == nil {
		return
	}

	currentMaker := level.Head
	for currentMaker != nil && takeOrder.RemainingQuantity > 0 {
		tradeQty := min(takeOrder.RemainingQuantity, currentMaker.RemainingQuantity)

		takeOrder.RemainingQuantity -= tradeQty
		currentMaker.RemainingQuantity -= tradeQty
		level.TotalVol -= int(tradeQty)

		if currentMaker.RemainingQuantity == 0 {
			currentMaker.Status = StatusFilled
			ob.removeOrderInternal(currentMaker)
			currentMaker = level.Head
		} else {
			currentMaker.Status = StatusPartial
		}
	}

	if takeOrder.RemainingQuantity == 0 {
		takeOrder.Status = StatusFilled
	} else {
		takeOrder.Status = StatusPartial
	}
}