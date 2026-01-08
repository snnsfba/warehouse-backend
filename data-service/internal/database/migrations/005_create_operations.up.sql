CREATE TABLE operations(
    operation_id SERIAL PRIMARY KEY,
    product_id INTEGER NOT NULL,
    order_id INTEGER,
    operation_type VARCHAR(20) NOT NULL CHECK (operation_type IN ('incoming', 'outgoing')),
    change_quant INTEGER NOT NULL CHECK (change_quant != 0),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (product_id) REFERENCES products(product_id),
    FOREIGN KEY (order_id) REFERENCES orders(order_id)
);