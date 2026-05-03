-- name: CreateOrder :one
INSERT INTO orders (
    id, user_id, price, quantity, side, remaining_quantity, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders
WHERE id = $1 LIMIT 1;

-- name: ListActiveOrdersBySide :many
SELECT * FROM orders
WHERE side = $1 AND remaining_quantity > 0 AND status IN ('PLACED', 'PARTIAL')
ORDER BY
    CASE WHEN $1 = 'BUY' THEN price END DESC,
    CASE WHEN $1 = 'SELL' THEN price END ASC,
    created_at ASC;

-- name: UpdateOrderQuantity :one
UPDATE orders
SET remaining_quantity = $2,
    status = $3
WHERE id = $1
RETURNING *;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET status = $2
WHERE id = $1;

-- name: UpdateOrderStatusesBatch :exec
UPDATE orders
SET status = val.status::order_status
FROM (
    SELECT
        unnest(@ids::uuid[]) as id,
        unnest(@statuses::text[]) as status
) as val
WHERE orders.id = val.id;

-- name: CreateTrade :one
INSERT INTO trades (
    id, buyer_order_id, seller_order_id, taker_user_id, maker_user_id, execution_price, quantity
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (id) DO NOTHING
RETURNING id;

-- name: CreateTradesBatch :exec
INSERT INTO trades (
    id, buyer_order_id, seller_order_id, taker_user_id, maker_user_id, execution_price, quantity
)
SELECT
    unnest(@ids::uuid[]),
    unnest(@buyer_order_ids::uuid[]),
    unnest(@seller_order_ids::uuid[]),
    unnest(@taker_user_ids::uuid[]),
    unnest(@maker_user_ids::uuid[]),
    unnest(@execution_prices::bigint[]),
    unnest(@quantities::bigint[])
ON CONFLICT (id) DO NOTHING;

-- name: GetRecentTrades :many
SELECT * FROM trades
ORDER BY executed_at DESC
LIMIT $1;

-- name: CancelOrder :exec
UPDATE orders
SET status = 'CANCELLED'
WHERE id = $1 AND status IN ('NEW', 'PLACED', 'PARTIAL');