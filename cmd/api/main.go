package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"golang.org/x/sync/errgroup"

	kafka_adapter "simple-orderbook/internal/adapters/kafka"
	metrics_adapter "simple-orderbook/internal/adapters/metrics"
	redis_adapter "simple-orderbook/internal/adapters/redis"
	"simple-orderbook/internal/adapters/repository"
	"simple-orderbook/internal/adapters/ws"
	"simple-orderbook/internal/api"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/ports"
	"simple-orderbook/internal/core/services"
	"simple-orderbook/internal/database"
	"simple-orderbook/internal/db"
)

type config struct {
	DBURL        string
	RedisURL     string
	KafkaURL     string
	KafkaTopic   string
	KafkaGroupID string
	JWTSecret    string
	JWTTTL       time.Duration
	Port         string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig() config {
	return config{
		DBURL:        getEnv("DB_URL", "postgres://postgres:password@db:5432/exchange?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis:6379"),
		KafkaURL:     getEnv("KAFKA_URL", "redpanda:9092"),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "order.created"),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "order-matching-group"),
		JWTSecret:    getEnv("JWT_SECRET", "change-me"),
		JWTTTL:       15 * time.Minute,
		Port:         ":8000",
	}
}

func createTopic(ctx context.Context, brokerAddr string, topic string) error {
	conn, err := kafka.DialContext(ctx, "tcp", brokerAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})

	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("creating topic %s: %w", topic, err)
	}

	return nil
}

func runMigrations(connStr string) error {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	migtationDir := "./migrations"

	err = goose.SetDialect("postgres")
	if err != nil {
		return err
	}

	err = goose.Up(db, migtationDir)
	return err
}

func setupRouter(orderSvc *services.OrderService, authSvc *services.AuthService, limiter ports.RateLimiter, jwtSecret string, wsHandler func(http.ResponseWriter, *http.Request), metrics ports.Metrics) http.Handler {
	mux := http.NewServeMux()

	orderHandler := api.NewOrderHandler(orderSvc, metrics)
	authHandler := api.NewAuthHandler(authSvc)

	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.Handle("GET /metrics", promhttp.Handler())

	protected := api.JWTMiddleware(jwtSecret)
	limited := api.RateLimitMiddleware(limiter, func(r *http.Request) string {
		id, _ := r.Context().Value(api.UserIDKey).(uuid.UUID)
		return id.String()
	})

	mux.Handle("POST /order", protected(limited(http.HandlerFunc(orderHandler.CreateOrder))))
	mux.Handle("GET /orderbook", protected(limited(http.HandlerFunc(orderHandler.GetOrderBook))))
	mux.Handle("/ws", protected(http.HandlerFunc(wsHandler)))

	return api.CORSMiddleware(mux)
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

	if err := runMigrations(cfg.DBURL); err != nil {
		return err
	}

	redisClient := goredis.NewClient(&goredis.Options{Addr: cfg.RedisURL})
	defer redisClient.Close()

	store := db.NewStore(pool)
	orderRepo := repository.NewPostgresOrderRepository(store)
	tradeRepo := repository.NewPostgresTradeRepository(store)
	outboxRepo := repository.NewPostgresOutboxRepository(store)
	authRepo := repository.NewPostgresAuthRepository(store)

	orderBook := domain.NewOrderBook()
	cache := redis_adapter.NewOrderBookRedisCache(redisClient)
	limiter := redis_adapter.NewFixedWindowRateLimiter(redisClient, 100, time.Minute)

	publisher := kafka_adapter.NewKafkaWriter(cfg.KafkaURL, cfg.KafkaTopic)
	reader := kafka_adapter.NewKafkaReader(cfg.KafkaURL, cfg.KafkaTopic, cfg.KafkaGroupID)

	for i := 0; i < 10; i++ {
		err = createTopic(ctx, cfg.KafkaURL, cfg.KafkaTopic)
		if err == nil {
			slog.Info("Kafka topic verified/created", "topic", cfg.KafkaTopic)
			break
		}
		slog.Warn("Waiting for Kafka/Redpanda", "attempt", i+1, "error", err)
		time.Sleep(4 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to create kafka topic after retries: %w", err)
	}

	authSvc := services.NewAuthService(authRepo, []byte(cfg.JWTSecret), cfg.JWTTTL)
	relay := services.NewOutboxRelay(outboxRepo, publisher)

	promMetrics := metrics_adapter.NewPrometheusMetrics()

	wsHub := ws.NewHub(redisClient)

	worker := services.NewOrderWorker(reader, orderBook, cache, orderRepo, tradeRepo, promMetrics, wsHub, store)

	orderSvc := services.NewOrderService(store, orderRepo, outboxRepo, orderBook, cache, relay)

	if err := orderSvc.RebuildOrderBook(ctx); err != nil {
		return fmt.Errorf("failed to rebuild order book: %w", err)
	}

	mux := setupRouter(orderSvc, authSvc, limiter, cfg.JWTSecret, wsHub.HandleWebSocket, promMetrics)

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
		slog.Info("Starting WS Hub")
		wsHub.Run(gCtx)
		return nil
	})

	g.Go(func() error {
		slog.Info("Starting worker")
		worker.Run(gCtx)
		return nil
	})

	g.Go(func() error {
		slog.Info("Starting server", "port", cfg.Port)
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
