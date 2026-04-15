package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
	metrics_adapter "simple-orderbook/internal/adapters/metrics"
	redis_adapter "simple-orderbook/internal/adapters/redis"
	"simple-orderbook/internal/adapters/repository"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/services"
	"simple-orderbook/internal/database"
	"simple-orderbook/internal/db"
)

func runMigrationsTest(t *testing.T, connStr string) {
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	defer db.Close()

	migtationDir := "../../sql/migrations"

	err = goose.SetDialect("postgres")
	require.NoError(t, err)

	err = goose.Up(db, migtationDir)
	require.NoError(t, err)
}

func cleanEnvironment(t *testing.T, pool *pgxpool.Pool, ob *domain.OrderBook, rdb *goredis.Client) {
	ctx := context.Background()

	_, err := pool.Exec(ctx, "TRUNCATE orders, trades, outbox CASCADE")
	require.NoError(t, err)

	ob.Reset()
	rdb.FlushAll(ctx)
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

	runMigrationsTest(t, pgConnStr)

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

	promMetrics := metrics_adapter.NewPrometheusMetrics()

	subscriber := kafka_adapter.NewKafkaReader(kafkaAddr, "order.created", "order-matching-group")
	worker := services.NewOrderWorker(subscriber, orderBook, cache, orderRepo, tradeRepo, promMetrics, store)
	go worker.Run(ctx)

	jwtSecret := "test-secret"
	orderSvc := services.NewOrderService(store, orderRepo, outboxRepo, orderBook, cache, relay)
	authSvc := services.NewAuthService(authRepo, []byte(jwtSecret), 15*time.Minute)

	mux := setupRouter(orderSvc, authSvc, limiter, jwtSecret, promMetrics)

	t.Run("Register Login and Order", func(t *testing.T) {
		cleanEnvironment(t, pool, orderBook, rdb)

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

			if len(snapshot.Bids) != 1 || snapshot.Bids[0].Price != 50000 || snapshot.Bids[0].TotalVol != 10 {
				return false
			}

			return true
		}, 10*time.Second, 10*time.Millisecond)
	})

	t.Run("Measure Placement and Matching Latency", func(t *testing.T) {
		cleanEnvironment(t, pool, orderBook, rdb)

		regBody, _ := json.Marshal(map[string]string{"email": "trader1@test.com", "password": "hash"})
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("POST", "/register", bytes.NewBuffer(regBody)))

		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("POST", "/login", bytes.NewBuffer(regBody)))
		var loginResp struct{ Token string }
		json.Unmarshal(rr.Body.Bytes(), &loginResp)

		orderBody, _ := json.Marshal(map[string]interface{}{
			"price": 50000, "quantity": 10, "side": "buy",
		})

		startPlacement := time.Now()

		req := httptest.NewRequest("POST", "/order", bytes.NewBuffer(orderBody))
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code)

		var placementDuration time.Duration
		assert.Eventually(t, func() bool {
			var status string
			err := pool.QueryRow(ctx, "SELECT status FROM orders WHERE side='BUY' LIMIT 1").Scan(&status)
			if err != nil {
				return false
			}

			if status == "PLACED" {
				placementDuration = time.Since(startPlacement)
				return true
			}
			return false
		}, 10*time.Second, 10*time.Millisecond)

		slog.Info("Latency Result", "step", "placement", "duration", placementDuration)
	})

	t.Run("Full Match: Buy and Sell equal quantity", func(t *testing.T) {
		cleanEnvironment(t, pool, orderBook, rdb)

		regBody, _ := json.Marshal(map[string]string{"email": "matcher@gmail.com", "password": "hash"})
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("POST", "/register", bytes.NewBuffer(regBody)))

		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("POST", "/login", bytes.NewBuffer(regBody)))
		var loginResp struct{ Token string }
		json.Unmarshal(rr.Body.Bytes(), &loginResp)

		startPlacement := time.Now()

		buyBody, _ := json.Marshal(map[string]interface{}{"price": 50000, "quantity": 5, "side": "buy"})
		rrBuy := httptest.NewRecorder()
		reqBuy := httptest.NewRequest("POST", "/order", bytes.NewBuffer(buyBody))
		reqBuy.Header.Set("Authorization", "Bearer "+loginResp.Token)
		mux.ServeHTTP(rrBuy, reqBuy)

		var buyOrder struct{ ID uuid.UUID }
		json.Unmarshal(rrBuy.Body.Bytes(), &buyOrder)

		sellBody, _ := json.Marshal(map[string]interface{}{"price": 50000, "quantity": 5, "side": "sell"})
		rrSell := httptest.NewRecorder()
		reqSell := httptest.NewRequest("POST", "/order", bytes.NewBuffer(sellBody))
		reqSell.Header.Set("Authorization", "Bearer "+loginResp.Token)
		mux.ServeHTTP(rrSell, reqSell)

		var sellOrder struct{ ID uuid.UUID }
		json.Unmarshal(rrSell.Body.Bytes(), &sellOrder)

		assert.Eventually(t, func() bool {
			var buyStatus, sellStatus string

			errB := pool.QueryRow(ctx, "SELECT status FROM orders WHERE id=$1", buyOrder.ID).Scan(&buyStatus)
			errS := pool.QueryRow(ctx, "SELECT status FROM orders WHERE id=$1", sellOrder.ID).Scan(&sellStatus)

			var tradeCountBuy, tradeCountSell int

			errTB := pool.QueryRow(ctx, "SELECT count(*) FROM trades WHERE buyer_order_id=$1", buyOrder.ID).Scan(&tradeCountBuy)
			errTS := pool.QueryRow(ctx, "SELECT count(*) FROM trades WHERE seller_order_id=$1", sellOrder.ID).Scan(&tradeCountSell)

			if errB != nil || errS != nil || errTB != nil || errTS != nil {
				slog.Error("Query failed", "errB", errB, "errS", errS, "errTB", errTB, "errTS", errTS)
				return false
			}

			slog.Info("Matching Stats",
				"targetBuyID", buyOrder.ID,
				"buyStatus", buyStatus,
				"sellStatus", sellStatus,
				"buyTradesFound", tradeCountBuy,
				"sellTradesFound", tradeCountSell,
			)

			return buyStatus == "FILLED" && sellStatus == "FILLED" && tradeCountBuy == 1 && tradeCountSell == 1
		}, 15*time.Second, 10*time.Millisecond)

		slog.Info("Latency Result for Full Match", "step", "placement", "duration", time.Since(startPlacement))
	})

	t.Run("Verify Prometheus Metrics", func(t *testing.T) {
		cleanEnvironment(t, pool, orderBook, rdb)

		orderBody, _ := json.Marshal(map[string]interface{}{
			"price": 100, "quantity": 1, "side": "buy",
		})

		regBody, _ := json.Marshal(map[string]string{"email": "matcher@gmail.com", "password": "hash"})
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("POST", "/register", bytes.NewBuffer(regBody)))

		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("POST", "/login", bytes.NewBuffer(regBody)))
		var loginResp struct{ Token string }
		json.Unmarshal(rr.Body.Bytes(), &loginResp)

		req := httptest.NewRequest("POST", "/order", bytes.NewBuffer(orderBody))
		req.Header.Set("Authorization", "Bearer "+loginResp.Token)
		rr = httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		require.Equal(t, http.StatusCreated, rr.Code)

		assert.Eventually(t, func() bool {
			metricsRR := httptest.NewRecorder()
			mux.ServeHTTP(metricsRR, httptest.NewRequest("GET", "/metrics", nil))

			body := metricsRR.Body.String()

			hasPlacement := strings.Contains(body, "exchange_order_placement_latency_seconds_count")
			hasMatching := strings.Contains(body, "exchange_matching_engine_latency_seconds_count")
			has2E := strings.Contains(body, "exchange_order_e2e_latency_seconds_count")

			if hasPlacement && hasMatching && has2E {
				slog.Info("asd", "content", body)
				return true
			} else {
				return false
			}
		}, 10*time.Second, 200*time.Millisecond)
	})
}
