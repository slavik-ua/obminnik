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
	"github.com/gorilla/websocket"
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
	"simple-orderbook/internal/adapters/ws"
	"simple-orderbook/internal/core/domain"
	"simple-orderbook/internal/core/services"
	"simple-orderbook/internal/database"
	"simple-orderbook/internal/db"
	"simple-orderbook/internal/pkg/idgen"
)

// --- Test Helpers ---

func runMigrationsTest(t *testing.T, connStr string) {
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	defer db.Close()

	migrationDir := "../../sql/migrations"
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.Up(db, migrationDir))
}

func cleanEnvironment(t *testing.T, pool *pgxpool.Pool, worker *services.OrderWorker, rdb *goredis.Client) {
	ctx := context.Background()
	_, err := pool.Exec(ctx, "TRUNCATE users, orders, trades, outbox, balances, ledger_entries CASCADE")
	require.NoError(t, err)

	worker.Reset()
	rdb.FlushAll(ctx)
}

func registerAndLogin(t *testing.T, mux http.Handler, email string) string {
	regBody, _ := json.Marshal(map[string]string{"email": email, "password": "password123"})
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("POST", "/register", bytes.NewBuffer(regBody)))
	require.Equal(t, http.StatusCreated, rr.Code)

	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest("POST", "/login", bytes.NewBuffer(regBody)))
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct{ Token string }
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	return resp.Token
}

func deposit(t *testing.T, mux http.Handler, token, asset string, amount int64) {
	body, _ := json.Marshal(map[string]interface{}{"asset": asset, "amount": amount})
	req := httptest.NewRequest("POST", "/deposit", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
}

// --- Integration Tests ---

func TestFullApplicationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. Infrastructure Setup (Postgres, Redis, Redpanda)
	pgContainer, err := postgres.Run(ctx, "postgres:16",
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

	kafkaContainer, err := redpanda.Run(ctx, "docker.redpanda.com/redpandadata/redpanda:v23.2.1")
	require.NoError(t, err)
	defer kafkaContainer.Terminate(ctx)
	kafkaAddr, _ := kafkaContainer.KafkaSeedBroker(ctx)
	require.NoError(t, createTopic(ctx, kafkaAddr, "order.created"))

	// 2. Application Setup
	pool, _ := database.NewPostgresPool(ctx, pgConnStr)
	defer pool.Close()
	rdb := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	store := db.NewStore(pool)
	fastGen := idgen.NewGenerator(ctx, 2000)

	orderRepo := repository.NewPostgresOrderRepository(store)
	tradeRepo := repository.NewPostgresTradeRepository(store)
	accountRepo := repository.NewPostgresAccountRepository(store, fastGen)
	outboxRepo := repository.NewPostgresOutboxRepository(store)
	authRepo := repository.NewPostgresAuthRepository(store)

	orderBook := domain.NewOrderBook(fastGen)
	cache := redis_adapter.NewOrderBookRedisCache(rdb)
	limiter := redis_adapter.NewFixedWindowRateLimiter(rdb, 1000, time.Minute)
	publisher := kafka_adapter.NewKafkaWriter(kafkaAddr, "order.created")
	relay := services.NewOutboxRelay(outboxRepo, publisher)
	go relay.Run(ctx)

	promMetrics := metrics_adapter.NewPrometheusMetrics()
	subscriber := kafka_adapter.NewKafkaReader(kafkaAddr, "order.created", "order-matching-group")
	wsHub := ws.NewHub(rdb)
	go wsHub.Run(ctx)

	balanceCache := domain.NewBalanceCache()
	worker := services.NewOrderWorker(subscriber, orderBook, cache, orderRepo, tradeRepo, accountRepo, balanceCache, promMetrics, wsHub, store)
	go worker.Run(ctx)

	jwtSecret := "test-secret"
	orderSvc := services.NewOrderService(store, orderRepo, accountRepo, outboxRepo, orderBook, cache, relay, fastGen)
	authSvc := services.NewAuthService(authRepo, []byte(jwtSecret), 15*time.Minute)

	mux := setupRouter(orderSvc, authSvc, limiter, jwtSecret, wsHub.HandleWebSocket, promMetrics, fastGen)
	server := httptest.NewServer(mux)
	defer server.Close()

	// --- Test Cases ---

	t.Run("Standard Flow: Deposit -> Place Order -> Verify Status", func(t *testing.T) {
		cleanEnvironment(t, pool, worker, rdb)
		token := registerAndLogin(t, mux, "trader@test.com")

		deposit(t, mux, token, "USD", 1000000*domain.Decimals)
		time.Sleep(500 * time.Millisecond) // Wait for Kafka propagation

		orderBody, _ := json.Marshal(map[string]interface{}{
			"price": 50000 * domain.Decimals, "quantity": 10 * domain.Decimals, "side": "buy",
		})
		req := httptest.NewRequest("POST", "/order", bytes.NewBuffer(orderBody))
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusCreated, rr.Code)

		assert.Eventually(t, func() bool {
			var status string
			_ = pool.QueryRow(ctx, "SELECT status FROM orders LIMIT 1").Scan(&status)
			return status == "PLACED"
		}, 10*time.Second, 100*time.Millisecond)
	})

	t.Run("Trade Settlement: BTC/USD Exchange", func(t *testing.T) {
		cleanEnvironment(t, pool, worker, rdb)
		buyerToken := registerAndLogin(t, mux, "buyer@exchange.com")
		sellerToken := registerAndLogin(t, mux, "seller@exchange.com")

		deposit(t, mux, buyerToken, "USD", 50000*domain.Decimals)
		deposit(t, mux, sellerToken, "BTC", 1*domain.Decimals)
		time.Sleep(500 * time.Millisecond)

		// Place matching orders
		buyBody, _ := json.Marshal(map[string]interface{}{"price": 50000 * domain.Decimals, "quantity": 1 * domain.Decimals, "side": "buy"})
		reqBuy := httptest.NewRequest("POST", "/order", bytes.NewBuffer(buyBody))
		reqBuy.Header.Set("Authorization", "Bearer "+buyerToken)
		mux.ServeHTTP(httptest.NewRecorder(), reqBuy)

		sellBody, _ := json.Marshal(map[string]interface{}{"price": 50000 * domain.Decimals, "quantity": 1 * domain.Decimals, "side": "sell"})
		reqSell := httptest.NewRequest("POST", "/order", bytes.NewBuffer(sellBody))
		reqSell.Header.Set("Authorization", "Bearer "+sellerToken)
		mux.ServeHTTP(httptest.NewRecorder(), reqSell)

		assert.Eventually(t, func() bool {
			var buyerBTC, sellerUSD int64
			_ = pool.QueryRow(ctx, "SELECT available FROM balances WHERE asset_symbol='BTC' AND user_id IN (SELECT id FROM users WHERE email='buyer@exchange.com')").Scan(&buyerBTC)
			_ = pool.QueryRow(ctx, "SELECT available FROM balances WHERE asset_symbol='USD' AND user_id IN (SELECT id FROM users WHERE email='seller@exchange.com')").Scan(&sellerUSD)
			return buyerBTC == 1*domain.Decimals && sellerUSD == 50000*domain.Decimals
		}, 10*time.Second, 100*time.Millisecond)
	})

	t.Run("Order Cancellation and Fund Return", func(t *testing.T) {
		cleanEnvironment(t, pool, worker, rdb)
		token := registerAndLogin(t, mux, "canceller@test.com")
		deposit(t, mux, token, "USD", 1000000*domain.Decimals)
		time.Sleep(500 * time.Millisecond)

		// Place order for $500 (Price 50, Quantity 10)
		orderBody, _ := json.Marshal(map[string]interface{}{"price": 50 * domain.Decimals, "quantity": 10 * domain.Decimals, "side": "buy"})
		reqOrder := httptest.NewRequest("POST", "/order", bytes.NewBuffer(orderBody))
		reqOrder.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, reqOrder)
		require.Equal(t, http.StatusCreated, rr.Code)

		var order struct{ ID uuid.UUID }
		json.Unmarshal(rr.Body.Bytes(), &order)

		// 1. Verify $500 is locked in DB (Wait for Worker)
		assert.Eventually(t, func() bool {
			var locked, avail int64
			err := pool.QueryRow(ctx, "SELECT available, locked FROM balances WHERE asset_symbol='USD' AND user_id IN (SELECT id FROM users WHERE email='canceller@test.com')").Scan(&avail, &locked)
			if err != nil {
				slog.Warn("Test Diagnostic: Balance query failed", "error", err)
				return false
			}
			slog.Info("Test Diagnostic: Balance state", "avail", avail, "locked", locked)
			return locked > 0
		}, 10*time.Second, 500*time.Millisecond)

		// 2. Cancel order
		cancelBody, _ := json.Marshal(map[string]interface{}{"order_id": order.ID})
		reqCancel := httptest.NewRequest("POST", "/order/cancel", bytes.NewBuffer(cancelBody))
		reqCancel.Header.Set("Authorization", "Bearer "+token)
		rrCancel := httptest.NewRecorder()
		mux.ServeHTTP(rrCancel, reqCancel)
		require.Equal(t, http.StatusOK, rrCancel.Code)

		assert.Eventually(t, func() bool {
			var avail int64
			err := pool.QueryRow(ctx, "SELECT available FROM balances WHERE asset_symbol='USD' AND user_id IN (SELECT id FROM users WHERE email='canceller@test.com')").Scan(&avail)
			if err != nil {
				return false
			}
			return avail == 1000000*domain.Decimals
		}, 10*time.Second, 100*time.Millisecond)
	})

	t.Run("WebSocket: Live Depth Updates", func(t *testing.T) {
		cleanEnvironment(t, pool, worker, rdb)
		token := registerAndLogin(t, mux, "ws-user@test.com")

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
		header := http.Header{}
		header.Add("Authorization", "Bearer "+token)
		wsConn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
		require.NoError(t, err)
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		defer wsConn.Close()

		deposit(t, mux, token, "USD", 1000*domain.Decimals)
		time.Sleep(200 * time.Millisecond)

		orderBody, _ := json.Marshal(map[string]interface{}{"price": 100 * domain.Decimals, "quantity": 1 * domain.Decimals, "side": "buy"})
		req := httptest.NewRequest("POST", "/order", bytes.NewBuffer(orderBody))
		req.Header.Set("Authorization", "Bearer "+token)
		mux.ServeHTTP(httptest.NewRecorder(), req)

		foundUpdate := false
		for i := 0; i < 5; i++ {
			wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, message, _ := wsConn.ReadMessage()
			if strings.Contains(string(message), "ORDERBOOK_UPDATE") {
				foundUpdate = true
				break
			}
		}
		assert.True(t, foundUpdate)
	})

	t.Run("Observability: Metrics Endpoint", func(t *testing.T) {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest("GET", "/metrics", http.NoBody))
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "exchange_order_placement_latency_seconds")
	})
}
