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

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"

	"simple-orderbook/internal/adapters/redis"
	"simple-orderbook/internal/adapters/kafka"
	"simple-orderbook/internal/adapters/repository"
	"simple-orderbook/internal/api"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/services"
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
	outboxRepo := repository.NewPostgresOutboxRepository(store)

	publisher := kafka.NewKafkaPublisher(os.Getenv("KAFKA_ADDR"))
	relay := services.NewOutboxRelay(outboxRepo, publisher)

	relayCtx, relayCancel := context.WithCancel(context.Background())
	defer relayCancel()
	go relay.Run(relayCtx)

	redisClient := goredis.NewClient(&goredis.Options{
		Addr: os.Getenv("REDIS_URL"),
	})
	defer redisClient.Close()

	cache := redis.NewOrderBookRedisCache(redisClient)

	limiter := redis.NewFixedWindowRateLimiter(redisClient, 100, time.Minute)

	svc := services.NewOrderService(store, orderRepo, tradeRepo, outboxRepo, orderBook, cache)
	handler := api.NewOrderHandler(svc)

	if err := svc.RebuildOrderBook(context.Background()); err != nil {
		log.Fatalf("failed to rebuild order book: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	jwtTTL := 15 * time.Minute

	authRepo := repository.NewPostgresAuthRepository(store)
	authService := services.NewAuthService(authRepo, []byte(jwtSecret), jwtTTL)
	authHandler := api.NewAuthHandler(authService)

	mux := http.NewServeMux()

	// finalHandler := api.RateLimitMiddleware(limiter, api.IPKey)(http.HandlerFunc(handler.CreateOrder))

	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	protected := api.JWTMiddleware(jwtSecret)
	limited := api.RateLimitMiddleware(limiter, func(r *http.Request) string {
		id, _ := r.Context().Value(api.UserIDKey).(uuid.UUID)
		return id.String()
	})

	mux.Handle("POST /order", protected(limited(http.HandlerFunc(handler.CreateOrder))))
	mux.Handle("GET /orderbook", protected(limited(http.HandlerFunc(handler.GetOrderBook))))
	// mux.Handle("GET /orderbook", api.RateLimitMiddleware(limiter, api.IPKey)(http.HandlerFunc(handler.GetOrderBook)))

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
