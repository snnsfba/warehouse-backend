package models

import "time"

type Product struct {
	ProductID   int       `json:"product_id"`
	Price       float64   `json:"price"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Quantity    int       `json:"quantity"`
	Category    string    `json:"category"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Customer struct {
	Name         string    `json:"name"`
	CustomerID   int       `json:"customer_id"`
	PhoneNumber  string    `json:"phone_number"`
	Address      string    `json:"address"`
	Email        string    `json:"email"`
	RegisteredAt time.Time `json:"registered_at"`
}

type Order struct {
	OrderID     int       `json:"order_id"`
	TotalAmount float64   `json:"total_amount"`
	Status      string    `json:"status"`
	CustomerID  int       `json:"customer_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type OrderItem struct {
	OrderItemID int     `json:"order_item_id"`
	OrderID     int     `json:"order_id"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
	ProductID   int     `json:"product_id"`
}

type Operation struct {
	OperationID   int       `json:"operation_id"`
	ProductID     int       `json:"product_id"`
	OrderID       int       `json:"order_id"`
	OperationType string    `json:"operation_type"`
	ChangeQuant   int       `json:"change_quant"`
	CreatedAt     time.Time `json:"created_at"`
}
