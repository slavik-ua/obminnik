-- +goose UP
CREATE TABLE balances (
    user_id UUID NOT NULL,
    asset_symbol VARCHAR(10) NOT NULL,
    available BIGINT NOT NULL DEFAULT 0,
    locked BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, asset_symbol),
    CONSTRAINT positive_balances CHECK (available >= 0 AND locked >= 0)
);

CREATE TABLE ledger_entries (
    id UUID NOT NULL,
    user_id UUID NOT NULL,
    asset_symbol VARCHAR(10) NOT NULL,
    amount BIGINT NOT NULL,
    balance_type VARCHAR(10) NOT NULL, -- 'AVAILABLE' or 'LOCKED'
    reference_type VARCHAR(20) NOT NULL, -- 'ORDER_LOCK', 'TRADE_SETTLEMENT', 'WITHDRAWL'
    reference_id UUID NOT NULL, -- OrderID or TradeID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ledger_user_asset ON ledger_entries(user_id, asset_symbol);

-- +goose DOWN
DROP INDEX IF EXISTS idx_ledger_user_asset;
DROP TABLE IF EXISTS ledger_entries;
DROP TABLE IF EXISTS balances;