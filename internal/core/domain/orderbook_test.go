package domain

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestOrderBook_Matching(t *testing.T) {
	userID := uuid.New()

	t.Run("Full match", func(t *testing.T) {
		ob := NewOrderBook()
		ob.PlaceOrder(uuid.New(), userID, 100, 10, SideBuy, nil)

		trades, status := ob.PlaceOrder(uuid.New(), userID, 100, 10, SideSell, nil)

		if len(trades) != 1 {
			t.Fatalf("expected 1 trade, got %d", len(trades))
		}
		if status != StatusFilled {
			t.Fatalf("expected taker status 3, got %d", status)
		}

		trade := trades[0]
		if trade.Price != 100 || trade.Quantity != 10 {
			t.Errorf("incorrect trade execution: %+v", trade)
		}
	})

	t.Run("Partial Fill: Taker smaller than Maker", func(t *testing.T) {
		ob := NewOrderBook()
		makerID := uuid.New()

		ob.PlaceOrder(makerID, userID, 100, 10, SideBuy, nil)

		trades, status := ob.PlaceOrder(uuid.New(), userID, 100, 5, SideSell, nil)

		if len(trades) != 1 {
			t.Fatalf("expected 1 trade, got %d", len(trades))
		}
		if status != StatusFilled {
			t.Errorf("expected taker to be 3, got %d", status)
		}
		if trades[0].Quantity != 5 {
			t.Errorf("expected partial fill of 5, got %d", trades[0].Quantity)
		}

		maker, ok := ob.GetOrder(makerID)
		if !ok || maker.RemainingQuantity != 5 {
			t.Errorf("maker should have 5 remaining, got %d", maker.RemainingQuantity)
		}
	})

	t.Run("Patial Fill: Taker larger than Maker", func(t *testing.T) {
		ob := NewOrderBook()
		ob.PlaceOrder(uuid.New(), userID, 100, 10, SideBuy, nil)

		trades, status := ob.PlaceOrder(uuid.New(), userID, 100, 15, SideSell, nil)

		if len(trades) != 1 {
			t.Fatalf("expected 1 trade, got %d", len(trades))
		}
		if status != StatusPartial {
			t.Errorf("expected taker status 2, got %d", status)
		}
		if trades[0].Quantity != 10 {
			t.Fatalf("expected trade quantity 10, got %d", trades[0].Quantity)
		}
	})

	t.Run("Price-Time Priority", func(t *testing.T) {
		ob := NewOrderBook()
		userA := uuid.New()
		userB := uuid.New()
		
		idA := uuid.New()
		idB := uuid.New()

		ob.PlaceOrder(idA, userA, 100, 10, SideBuy, nil)
		ob.PlaceOrder(idB, userB, 100, 10, SideBuy, nil)

		trades, _ := ob.PlaceOrder(uuid.New(), userID, 100, 10, SideSell, nil)

		if trades[0].MakerOrderID != idA {
			t.Errorf("expected to match with first order %s, matched with %s", idA, trades[0].MakerOrderID)
		}
		
		_, ok := ob.GetOrder(idB)
		if !ok {
			t.Error("second order should still be in the book")
		}
	})

	t.Run("Multiple Price Levels", func(t *testing.T) {
		ob := NewOrderBook()

		ob.PlaceOrder(uuid.New(), userID, 100, 10, SideBuy, nil)
		ob.PlaceOrder(uuid.New(), userID, 90, 10, SideBuy, nil)

		trades, status := ob.PlaceOrder(uuid.New(), userID, 90, 15, SideSell, nil)

		if len(trades) != 2 {
			t.Fatalf("expected 2 trades, got %d", len(trades))
		}
		if status != StatusFilled {
			t.Errorf("expected taker to be fully filled, got %d", status)
		}
		if trades[0].Price != 100 || trades[1].Price != 90 {
			t.Error("trades executed in wrong price priority")
		}
	})

	t.Run("No match (Price too high)", func(t *testing.T) {
		ob := NewOrderBook()
		ob.PlaceOrder(uuid.New(), userID, 100, 10, SideBuy, nil)

		trades, status := ob.PlaceOrder(uuid.New(), userID, 101, 10, SideSell, nil)

		if len(trades) != 0 {
			t.Error("orders should not have matched")
		}
		if status != StatusPlaced {
			t.Errorf("expected status 1, got %d", status)
		}
	})
}

func TestOrderBook_Concurrency(t *testing.T) {
	ob := NewOrderBook()
	const workers = 100
	const ordersPerWorker = 10
	const targetPrice = 100

	var wg sync.WaitGroup
	wg.Add(workers * 2)

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < ordersPerWorker; j++ {
				ob.PlaceOrder(uuid.New(), uuid.New(), targetPrice, 1, SideBuy, nil)
			}
		}(i)
	}

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < ordersPerWorker; j++ {
				ob.PlaceOrder(uuid.New(), uuid.New(), targetPrice, 1, SideSell, nil)
			}
		}(i)
	}

	wg.Wait()

	snapshot := ob.Snapshot()

	if len(snapshot.Bids) != 0 || (len(snapshot.Asks) != 0) {
		t.Errorf("expected empty book, got Bids: %d, Asks: %d", len(snapshot.Bids), len(snapshot.Asks))
	}
}
