-- getting orders by user
CREATE TABLE IF NOT EXISTS orders_by_user (
    customer_id     BIGINT,
    id              BIGINT,
    status          TEXT,
    total_price     INT,
    created_at      TIMESTAMP,
    PRIMARY KEY (customer_id, created_at, id)
) WITH CLUSTERING ORDER BY (created_at DESC, id DESC);

-- Table for storing order items by order ID
CREATE TABLE IF NOT EXISTS order_items_by_user (
    order_id        BIGINT,
    product_id      BIGINT,
    quantity        INT,
    price_per_unit  DOUBLE,
    PRIMARY KEY (order_id, product_id)
) WITH CLUSTERING ORDER BY (product_id ASC);

-- Table for storing product catalog
CREATE TABLE IF NOT EXISTS products (
    id              BIGINT PRIMARY KEY,
    name            TEXT,
    description     TEXT,
    created_at      TIMESTAMP
);