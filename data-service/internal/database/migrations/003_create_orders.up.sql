CREATE TABLE orders(
    order_id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL,
    total_amount DECIMAL(15,2) NOT NULL,
    status VARCHAR(155) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (customer_id) REFERENCES customers(customer_id)
);