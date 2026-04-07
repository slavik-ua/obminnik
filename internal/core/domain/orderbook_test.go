package domain

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestOrderBook_Matching(t *testing.T) {
	ob := NewOrderBook()
	userID := uuid.New()

	t.Run("Full match", func(t *testing.T) {
		ob.PlaceOrder(uuid.New(), userID, 100, 10, SideBuy, nil)

		trades := ob.PlaceOrder(uuid.New(), userID, 100, 10, SideSell, nil)

		if len(trades) != 1 {
			t.Fatalf("expected 1 trade, got %d", len(trades))
		}

		trade := trades[0]
		if trade.Price != 100 || trade.Quantity != 10 {
			t.Errorf("incorrect trade execution: %+v", trade)
		}
	})

	t.Run("Partial Fill", func(t *testing.T) {
		ob := NewOrderBook()

		ob.PlaceOrder(uuid.New(), userID, 100, 10, SideBuy, nil)

		trades := ob.PlaceOrder(uuid.New(), userID, 100, 5, SideSell, nil)

		if len(trades) != 1 {
			t.Fatalf("expected 1 trade, got %d", len(trades))
		}
		if trades[0].Quantity != 5 {
			t.Errorf("expected partial fill of 5, got %d", trades[0].Quantity)
		}
	})

	t.Run("No match (Price too high)", func(t *testing.T) {
		ob := NewOrderBook()
		ob.PlaceOrder(uuid.New(), userID, 100, 10, SideBuy, nil)

		trades := ob.PlaceOrder(uuid.New(), userID, 101, 10, SideSell, nil)

		if len(trades) != 0 {
			t.Error("orders should not have matched")
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
