-- +goose UP
ALTER TABLE orders
    ALTER COLUMN price TYPE BIGINT
    USING (price * 10000)::BIGINT;

ALTER TABLE trades
    ALTER COLUMN execution_price TYPE BIGINT
    USING (execution_price * 10000)::BIGINT;

-- +goose DOWN
ALTER TABLE orders
    ALTER COLUMN price TYPE DECIMAL(18, 4)
    USING (price::DECIMAL / 10000);

ALTER TABLE trades
    ALTER COLUMN execution_price TYPE DECIMAL(18, 4)
    USING (execution_price::DECIMAL / 10000);