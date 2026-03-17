package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

	"github.com/joho/godotenv"

	"simple-orderbook/internal/adapters/repository"
	"simple-orderbook/internal/api"
	"simple-orderbook/internal/database"
	"simple-orderbook/internal/db"
)

type Order struct {
	ID       string
	Price    float64 // Change to other
	Quantity int
	Side     string
}

type OrderBook struct {
	Bids []Order
	Asks []Order
}

func NewOrderBook() *OrderBook {
	return &OrderBook{}
}

func (ob *OrderBook) AddOrder(order Order) {
	if order.Side == "BUY" {
		ob.Bids = append(ob.Bids, order)
		sort.Slice(ob.Bids, func(i, j int) bool {
			return ob.Bids[i].Price > ob.Bids[j].Price
		})
	} else {
		ob.Asks = append(ob.Asks, order)
		sort.Slice(ob.Asks, func(i, j int) bool {
			return ob.Asks[i].Price < ob.Asks[j].Price
		})
	}
}

func (ob *OrderBook) Match() {
	for len(ob.Bids) > 0 && len(ob.Asks) > 0 {
		bestBid := &ob.Bids[0]
		bestAsk := &ob.Asks[0]

		if bestBid.Price >= bestAsk.Price {
			fillQty := min(bestBid.Quantity, bestAsk.Quantity)

			bestBid.Quantity -= fillQty
			bestAsk.Quantity -= fillQty

			fmt.Println("Matched:", fillQty, "at price", bestAsk.Price)

			if bestBid.Quantity == 0 {
				ob.Bids = ob.Bids[1:]
			}
			if bestAsk.Quantity == 0 {
				ob.Asks = ob.Asks[1:]
			}
		} else {
			break
		}
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Could not load .env file")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL is not set in .env file")
	}

	pool, err := database.NewPostgresPool(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Could create database pool: %s, %v", dbURL, err)
	}

	store := db.NewStore(pool)
	repo := repository.NewPostgresOrderRepository(store)
	handler := api.NewOrderHandler(repo)

	mux := http.NewServeMux()

	mux.HandleFunc("/order", handler.CreateOrder)

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,

		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	log.Println("Started listening at :8000")
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}

	// ob := NewOrderBook()

	// ob.AddOrder(Order{"ID1", 140, 20, "BUY"})
	// ob.AddOrder(Order{"ID2", 160, 20, "BUY"})
	// ob.AddOrder(Order{"ID3", 120, 20, "SELL"})

	// ob.Match()
}
