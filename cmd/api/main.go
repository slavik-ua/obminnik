package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"

	"simple-orderbook/internal/adapters/repository"
	"simple-orderbook/internal/adapters/redis"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/services"
	"simple-orderbook/internal/api"
	"simple-orderbook/internal/database"
	"simple-orderbook/internal/db"
)

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
	defer pool.Close()

	store := db.NewStore(pool)
	orderBook := domain.NewOrderBook()
	orderRepo := repository.NewPostgresOrderRepository(store)
	tradeRepo := repository.NewPostgresTradeRepository(store)

	redisClient := goredis.NewClient(&goredis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})
	defer redisClient.Close()

	cache := redis.NewOrderBookRedisCache(redisClient)

	limiter := redis.NewFixedWindowRateLimiter(redisClient, 100, time.Minute)

	svc := services.NewOrderService(store, orderRepo, tradeRepo, orderBook, cache)
	handler := api.NewOrderHandler(svc)

	if err := svc.RebuildOrderBook(context.Background()); err != nil {
		log.Fatalf("failed to rebuild order book: %v", err)
	}

	mux := http.NewServeMux()

	finalHandler := api.RateLimitMiddleware(limiter, api.IPKey)(http.HandlerFunc(handler.CreateOrder))
	mux.Handle("POST /order", finalHandler)

	mux.Handle("GET /orderbook", api.RateLimitMiddleware(limiter, api.IPKey)(http.HandlerFunc(handler.GetOrderBook)))

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,

		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Println("Started listening on :8000...")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting gracefully")
}
