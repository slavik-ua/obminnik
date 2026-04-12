package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	"github.com/testcontainers/testcontainers-go/wait"

	kafka_adapter "simple-orderbook/internal/adapters/kafka"
	redis_adapter "simple-orderbook/internal/adapters/redis"
	"simple-orderbook/internal/adapters/repository"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/services"
	"simple-orderbook/internal/database"
	"simple-orderbook/internal/db"
)

func runMigrations(t *testing.T, connStr string) {
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	defer db.Close()

	migtationDir := "../../sql/migrations"

	err = goose.SetDialect("postgres")
	require.NoError(t, err)

	err = goose.Up(db, migtationDir)
	require.NoError(t, err)
}

func TestFullApplicationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("exchange"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("pass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	defer pgContainer.Terminate(ctx)
	pgConnStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")

	runMigrations(t, pgConnStr)

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	defer redisContainer.Terminate(ctx)
	redisAddr, _ := redisContainer.Endpoint(ctx, "")

	kafkaContainer, err := redpanda.Run(ctx,
		"docker.redpanda.com/redpandadata/redpanda:v23.2.1",
	)
	require.NoError(t, err)
	defer kafkaContainer.Terminate(ctx)

	kafkaAddr, err := kafkaContainer.KafkaSeedBroker(ctx)
	require.NoError(t, err)

	err = createTopic(ctx, kafkaAddr, "order.created")
	require.NoError(t, err)

	time.Sleep(4 * time.Second)

	pool, err := database.NewPostgresPool(ctx, pgConnStr)
	require.NoError(t, err)
	defer pool.Close()

	rdb := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	store := db.NewStore(pool)

	orderRepo := repository.NewPostgresOrderRepository(store)
	tradeRepo := repository.NewPostgresTradeRepository(store)
	outboxRepo := repository.NewPostgresOutboxRepository(store)
	authRepo := repository.NewPostgresAuthRepository(store)

	orderBook := domain.NewOrderBook()
	cache := redis_adapter.NewOrderBookRedisCache(rdb)
	limiter := redis_adapter.NewFixedWindowRateLimiter(rdb, 100, time.Minute)
	publisher := kafka_adapter.NewKafkaWriter(kafkaAddr, "order.created")

	relay := services.NewOutboxRelay(outboxRepo, publisher)
	go relay.Run(ctx)

	subscriber := kafka_adapter.NewKafkaReader(kafkaAddr, "order.created", "order-matching-group")
	worker := services.NewOrderWorker(subscriber, orderBook, cache, orderRepo, tradeRepo, store)
	go worker.Run(ctx)

	jwtSecret := "test-secret"
	orderSvc := services.NewOrderService(store, orderRepo, outboxRepo, orderBook, cache, relay)
	authSvc := services.NewAuthService(authRepo, []byte(jwtSecret), 15*time.Minute)

	mux := setupRouter(orderSvc, authSvc, limiter, jwtSecret)

	slog.Info("BEFORE")
	time.Sleep(20 * time.Second)
	slog.Info("AFTER")

	t.Run("Register Login and Order", func(t *testing.T) {
		regBody, _ := json.Marshal(map[string]string{
			"email":    "test@test.com",
			"password": "hash",
		})

		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("POST", "/register", bytes.NewBuffer(regBody)))
		assert.Equal(t, http.StatusCreated, rr.Code)

		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("POST", "/login", bytes.NewBuffer(regBody)))
		var loginResp struct{ Token string }
		json.Unmarshal(rr.Body.Bytes(), &loginResp)

		orderBody, _ := json.Marshal(map[string]interface{}{
			"price": 50000, "quantity": 10, "side": "buy",
		})
		rr = httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/order", bytes.NewBuffer(orderBody))
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		mux.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)

		assert.Eventually(t, func() bool {
			var status string
			err := pool.QueryRow(ctx, "SELECT status FROM orders LIMIT 1").Scan(&status)
			if err != nil {
				return false
			}

			slog.Info("asd", "STATUS", status)

			return status == "PLACED"
		}, 15*time.Second, 10*time.Millisecond)

		assert.Eventually(t, func() bool {
			rr = httptest.NewRecorder()
			reqOB := httptest.NewRequest("GET", "/orderbook", nil)
			reqOB.Header.Set("Authorization", "Bearer "+loginResp.Token)
			mux.ServeHTTP(rr, reqOB)

			if rr.Code != http.StatusOK {
				return false
			}

			var snapshot struct {
				Bids []struct {
					Price    int64 `json:"price"`
					TotalVol int64 `json:"total_vol"`
				}
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &snapshot); err != nil {
				return false
			}

			if len(snapshot.Bids) != 1 && snapshot.Bids[0].Price != 50000 && snapshot.Bids[0].TotalVol != 10 {
				return false
			}

			return true
		}, 10*time.Second, 10*time.Millisecond)
	})
}
