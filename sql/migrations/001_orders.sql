-- +goose UP
CREATE TYPE order_side AS ENUM ('BUY', 'SELL');

CREATE TABLE orders (
    id UUID PRIMARY KEY,
    price DECIMAL(18, 4) NOT NULL,
    quantity INT NOT NULL,
    side order_side NOT NULL,
    remaining_quantity INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE trades (
    id SERIAL PRIMARY KEY,
    buyer_order_id UUID REFERENCES orders(id),
    seller_order_id UUID REFERENCES orders(id),
    execution_price DECIMAL(18, 4) NOT NULL,
    quantity INT NOT NULL,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose DOWN
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_side;