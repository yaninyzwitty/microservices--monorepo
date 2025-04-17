-- categories table for storing categories
CREATE TABLE IF NOT EXISTS categories (
    id BIGINT PRIMARY KEY,
    name STRING UNIQUE NOT NULL,
    description STRING,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS products (
    id BIGINT PRIMARY KEY,
    category_id BIGINT REFERENCES categories(id) NOT NULL,
    name STRING NOT NULL,
    description STRING,
    price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()

);

CREATE INDEX IF NOT EXISTS idx_products_category_id ON products(category_id);