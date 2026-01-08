CREATE TABLE products (
    product_id SERIAL PRIMARY KEY,
    price DECIMAL(10,2) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT,
    quantity INTEGER NOT NULL DEFAULT 0,
    category VARCHAR(200),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

