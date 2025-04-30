CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    first_name STRING NOT NULL,
    last_name STRING,
    email STRING NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
    
);