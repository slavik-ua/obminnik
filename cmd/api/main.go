package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	"simple-orderbook/internal/adapters/kafka"
	"simple-orderbook/internal/adapters/redis"
	"simple-orderbook/internal/adapters/repository"
	"simple-orderbook/internal/api"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/ports"
	"simple-orderbook/internal/core/services"
	"simple-orderbook/internal/database"
	"simple-orderbook/internal/db"
)

type config struct {
	DBURL     string
	RedisURL  string
	KafkaURL  string
	JWTSecret string
	JWTTTL    time.Duration
	Port      string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig() config {
	return config{
		DBURL:     getEnv("DB_URL", "localhost:5432"),
		RedisURL:  getEnv("REDIS_URL", "localhost:6379"),
		KafkaURL:  getEnv("KAFKA_URL", "localhost:9092"),
		JWTSecret: getEnv("JWT_SECRET", "change-me"),
		JWTTTL:    15 * time.Minute,
		Port:      ":8000",
	}
}

func setupRouter(orderSvc *services.OrderService, authSvc *services.AuthService, limiter ports.RateLimiter, jwtSecret string) *http.ServeMux {
	mux := http.NewServeMux()

	orderHandler := api.NewOrderHandler(orderSvc)
	authHandler := api.NewAuthHandler(authSvc)

	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)

	protected := api.JWTMiddleware(jwtSecret)
	limited := api.RateLimitMiddleware(limiter, func(r *http.Request) string {
		id, _ := r.Context().Value(api.UserIDKey).(uuid.UUID)
		return id.String()
	})

	mux.Handle("POST /order", protected(limited(http.HandlerFunc(orderHandler.CreateOrder))))
	mux.Handle("GET /orderbook", protected(limited(http.HandlerFunc(orderHandler.GetOrderBook))))

	return mux
}

func run() error {
	_ = godotenv.Load()
	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPostgresPool(ctx, cfg.DBURL)
	if err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	defer pool.Close()

	redisClient := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
	defer redisClient.Close()

	store := db.NewStore(pool)
	orderRepo := repository.NewPostgresOrderRepository(store)
	tradeRepo := repository.NewPostgresTradeRepository(store)
	outboxRepo := repository.NewPostgresOutboxRepository(store)
	authRepo := repository.NewPostgresAuthRepository(store)

	orderBook := domain.NewOrderBook()
	cache := redis.NewOrderBookRedisCache(redisClient)
	limiter := redis.NewFixedWindowRateLimiter(redisClient, 100, time.Minute)
	publisher := kafka.NewKafkaPublisher(cfg.KafkaURL)

	orderSvc := services.NewOrderService(store, orderRepo, tradeRepo, outboxRepo, orderBook, cache)
	authSvc := services.NewAuthService(authRepo, []byte(cfg.JWTSecret), cfg.JWTTTL)
	relay := services.NewOutboxRelay(outboxRepo, publisher)

	if err := orderSvc.RebuildOrderBook(ctx); err != nil {
		return fmt.Errorf("failed to rebuild order book: %w", err)
	}

	mux := setupRouter(orderSvc, authSvc, limiter, cfg.JWTSecret)

	server := &http.Server{
		Addr:              cfg.Port,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("Starting outbox relay")
		relay.Run(gCtx)
		return nil
	})

	g.Go(func() error {
		slog.Info("Starting server on %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		slog.Info("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	})

	return g.Wait()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}
