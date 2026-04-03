-- name: CreateOrder :one
INSERT INTO orders (
    id, price, quantity, side, remaining_quantity
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: ListActiveOrdersBySide :many
SELECT * FROM orders
WHERE side = $1 and remaining_quantity > 0
ORDER BY
    CASE WHEN $1 = 'BUY' THEN price END DESC,
    CASE WHEN $1 = 'SELL' THEN price END ASC,
    created_at ASC;

-- name: UpdateOrderQuantity :one
UPDATE orders
SET remaining_quantity = $2
WHERE id = $1
RETURNING *;

-- name: CreateTrade :one
INSERT INTO trades (
    id, buyer_order_id, seller_order_id, taker_user_id, maker_user_id, execution_price, quantity
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT ON CONSTRAINT trades_idempotency_key
DO NOTHING
RETURNING id;

-- name: GetRecentTrades :many
SELECT * FROM trades
ORDER BY executed_at DESC
LIMIT $1;

-- name: CancelOrder :exec
DELETE FROM orders WHERE id = $1;