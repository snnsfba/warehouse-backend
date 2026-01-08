CREATE TABLE customers (
    customer_id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    address TEXT,
    email VARCHAR(155) UNIQUE NOT NULL,
    registered_at TIMESTAMPTZ DEFAULT NOW()
);