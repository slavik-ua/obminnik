-- +goose UP
CREATE TYPE order_side AS ENUM ('BUY', 'SELL');

CREATE TABLE orders (
    id UUID PRIMARY KEY,
    price BIGINT NOT NULL,
    quantity BIGINT NOT NULL,
    side order_side NOT NULL,
    remaining_quantity BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE trades (
    id UUID PRIMARY KEY,
    buyer_order_id UUID REFERENCES orders(id),
    seller_order_id UUID REFERENCES orders(id),

    taker_user_id UUID NOT NULL,
    maker_user_id UUID NOT NULL,

    execution_price BIGINT NOT NULL,
    quantity BIGINT NOT NULL,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

    CONSTRAINT trades_idempontency_key UNIQUE (buyer_order_id, seller_order_id, execution_price, quantity)
);

-- +goose DOWN
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_side;