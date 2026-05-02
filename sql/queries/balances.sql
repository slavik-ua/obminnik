-- name: UpdateBalanceLock :one
UPDATE balances
SET
    available = available - @amount,
    locked = locked + @amount,
    updated_at = NOW()
WHERE user_id = @user_id
    AND asset_symbol = @asset_symbol
    AND available >= @amount
RETURNING available, locked;

-- name: UpdateBalanceUnlock :one
UPDATE balances
SET
    available = available + @amount,
    locked = locked - @amount,
    updated_at = NOW()
WHERE user_id = @user_id
    AND asset_symbol = @asset_symbol
    AND locked >= @amount
RETURNING available, locked;

-- name: CreateLedgerEntry :exec
INSERT INTO ledger_entries (
    id, user_id, asset_symbol, amount, balance_type, reference_type, reference_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: CreateLedgerEntries :copyfrom
INSERT INTO ledger_entries (
    id, user_id, asset_symbol, amount, balance_type, reference_type, reference_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
);

-- name: ListAllBalances :many
SELECT user_id, asset_symbol, available, locked
FROM balances;

-- name: EnsureBalancesExist :exec
INSERT INTO balances (user_id, asset_symbol, available, locked, updated_at)
SELECT
    unnest(@user_ids::uuid[]),
    unnest(@assets::text[]),
    0, 0, NOW()
ON CONFLICT (user_id, asset_symbol) DO NOTHING;

-- name: BatchUpdateBalances :exec
UPDATE balances AS b
SET
    available = b.available + updates.avail_delta,
    locked = b.locked + updates.lock_delta,
    updated_at = NOW()
FROM (
    SELECT
        unnest(@user_ids::uuid[]) as u_id,
        unnest(@assets::text[]) as a_sym,
        unnest(@avail_deltas::bigint[]) as avail_delta,
        unnest(@lock_deltas::bigint[]) as lock_delta
) AS updates
WHERE b.user_id = updates.u_id AND b.asset_symbol = updates.a_sym;

-- name: UpsertBalance :one
INSERT INTO balances (user_id, asset_symbol, available, locked, updated_at)
VALUES ($1, $2, $3, 0, NOW())
ON CONFLICT (user_id, asset_symbol)
DO UPDATE SET
    available = balances.available + EXCLUDED.available,
    updated_at = NOW()
RETURNING *;