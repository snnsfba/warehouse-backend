package repository

import (
	"context"
	"data-service/internal/models"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type orderRepo struct {
	db *pgx.Conn
}

func NewOrderRepository(db *pgx.Conn) OrderRepository {
	return &orderRepo{db: db}
}

type productStock struct {
	price    float64
	quantity int
}

func (r *orderRepo) CreateOrder(ctx context.Context, order *models.Order, items []models.OrderItem) error {
	if order == nil {
		return fmt.Errorf("%w: order cannot be nil", ErrInvalidInput)
	}

	if order.CustomerID <= 0 {
		return fmt.Errorf("%w: ID cannot be empty", ErrInvalidInput)
	}

	if len(items) == 0 {
		return fmt.Errorf("slice items cannot be empty: %w", ErrInvalidInput)
	}

	for _, item := range items {
		if item.Quantity <= 0 {
			return fmt.Errorf("quantity must be positive: %w", ErrInvalidInput)
		}
		if item.Price <= 0 {
			return fmt.Errorf("Price must be positive: %w", ErrInvalidInput)
		}
		if item.ProductID <= 0 {
			return fmt.Errorf("Product ID cannot be empty: %w", ErrInvalidInput)
		}
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	sql := ` SELECT 
		customer_id
		FROM customers WHERE customer_id = $1
	`

	var customer models.Customer

	err = tx.QueryRow(ctx, sql, order.CustomerID).Scan(&customer.CustomerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("failed to get customer by id: %w", err)
	}

	prosuctsIDs := []int{}
	for _, item := range items {
		prosuctsIDs = append(prosuctsIDs, item.ProductID)
	}

	sql2 := ` SELECT
	product_id,
	price,
	quantity
	FROM products WHERE product_id = ANY($1::int[])
	`

	var rows pgx.Rows
	rows, err = tx.Query(ctx, sql2, prosuctsIDs)
	if err != nil {
		return fmt.Errorf("failed to get products information: %w", err)
	}

	defer rows.Close()

	productInfo := make(map[int]struct {
		price    float64
		quantity int
	})

	for rows.Next() {
		var p models.Product
		err := rows.Scan(&p.ProductID,
			&p.Price,
			&p.Quantity,
		)
		if err != nil {
			return fmt.Errorf("failed to scan product data: %w", err)
		}

		productInfo[p.ProductID] = productStock{
			price:    p.Price,
			quantity: p.Quantity,
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to complete row iteration: %w", err)
	}

	for _, item := range items {
		info, exist := productInfo[item.ProductID]
		if !exist {
			return fmt.Errorf("product not found: %w", ErrNotFound)
		}
		if info.quantity < item.Quantity {
			return fmt.Errorf("%w: not enough in stock %d", ErrInvalidInput, item.ProductID)
		}
	}

	var total float64
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
	}
	order.TotalAmount = total

	insert := `INSERT INTO orders (
	customer_id,
	total_amount,
	status,
	created_at
	) VALUES ($1, $2, $3, $4)
	RETURNING order_id
	`

	err = tx.QueryRow(ctx, insert, order.CustomerID, order.TotalAmount, "created", time.Now()).Scan(&order.OrderID)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	for _, item := range items {
		insertItemSQL := `INSERT INTO order_items (order_id, product_id, quantity, price)
		VALUES ($1, $2, $3, $4)
	`
		_, err = tx.Exec(ctx, insertItemSQL, order.OrderID, item.ProductID, item.Quantity, item.Price)
		if err != nil {
			return fmt.Errorf("failed to create order item: %w", err)
		}

		update := ` UPDATE products SET quantity = quantity - $1 WHERE product_id = $2`

		result, err := tx.Exec(ctx, update, item.Quantity, item.ProductID)
		if err != nil {
			return fmt.Errorf("failed to update products %d: %w", item.ProductID, err)
		}

		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			return ErrNotFound
		}

		insertOperationSQL := ` INSERT INTO operations (
			product_id,
			order_id,
			operation_type,
			change_quant,
			created_at
		) VALUES ($1, $2, $3, $4, $5)
			`
		_, err = tx.Exec(ctx, insertOperationSQL, item.ProductID, order.OrderID, "outgoing", -item.Quantity, time.Now())
		if err != nil {
			return fmt.Errorf("failed to create operation: %w", err)
		}

	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *orderRepo) GetByID(ctx context.Context, id int) (*models.Order, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: order ID must be positive", ErrInvalidInput)
	}

	sql := ` SELECT
		order_id,
		customer_id,
		total_amount,
		status,
		created_at
		FROM orders 
		WHERE order_id = $1
	`

	var order models.Order

	err := r.db.QueryRow(ctx, sql, id).Scan(
		&order.OrderID,
		&order.CustomerID,
		&order.TotalAmount,
		&order.Status,
		&order.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get order %d: %w", id, err)
	}

	return &order, nil
}

func (r *orderRepo) GetAll(ctx context.Context) ([]models.Order, error) {
	sql := `
	SELECT 
		order_id,
		customer_id,
		total_amount,
		status,
		created_at
		FROM orders
		ORDER BY order_id`

	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to get all orders: %w", err)
	}

	defer rows.Close()

	var orders []models.Order

	for rows.Next() {
		var o models.Order

		err := rows.Scan(&o.OrderID,
			&o.CustomerID,
			&o.TotalAmount,
			&o.Status,
			&o.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan all orders: %w", err)
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to complete row iteration: %w", err)
	}

	return orders, nil
}

func (r *orderRepo) UpdateStatus(ctx context.Context, id int, status string) error {

	if status == "" {
		return fmt.Errorf("%w: Status cannot be empty", ErrInvalidInput)
	}

	validStatuses := map[string]bool{
		"created":   true,
		"paid":      true,
		"cancelled": true,
		"shipped":   true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("%w: invalid status '%s'", ErrInvalidInput, status)
	}

	sql := `UPDATE orders 
		SET status = $1
		WHERE order_id = $2
		`

	result, err := r.db.Exec(ctx, sql, status, id)
	if err != nil {
		return fmt.Errorf("update status order %d: %w", id, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *orderRepo) GetOrderWithItems(ctx context.Context, id int) (*models.Order, []models.OrderItem, error) {
	if id <= 0 {
		return nil, nil, fmt.Errorf("%w: order ID must be positive", ErrInvalidInput)
	}

	sql := `SELECT
	o.order_id,
	o.customer_id,
	o.total_amount,
	o.status,
	o.created_at,
	oi.order_item_id,
	oi.product_id,
	oi.quantity,
	oi.price
	FROM orders o
	LEFT JOIN order_items oi ON o.order_id = oi.order_id
	WHERE o.order_id = $1
	ORDER BY oi.order_item_id
	`

	rows, err := r.db.Query(ctx, sql, id)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get orders with items %d: %w", id, err)
	}

	defer rows.Close()

	var order *models.Order
	var items []models.OrderItem
	var orderFound bool

	for rows.Next() {
		var currentOrder models.Order
		var orderItemID pgtype.Int4 // вместо int
		var productID pgtype.Int4   // вместо int
		var quantity pgtype.Int4    // вместо int
		var price pgtype.Float4     // вместо float64

		err := rows.Scan(&currentOrder.OrderID,
			&currentOrder.CustomerID,
			&currentOrder.TotalAmount,
			&currentOrder.Status,
			&currentOrder.CreatedAt,
			&orderItemID,
			&productID,
			&quantity,
			&price,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("scan order/item: %w", err)
		}
		if !orderFound {
			order = &currentOrder
			orderFound = true
		}
		if orderItemID.Valid {
			items = append(items, models.OrderItem{
				OrderItemID: int(orderItemID.Int32),
				OrderID:     currentOrder.OrderID,
				ProductID:   int(productID.Int32),
				Quantity:    int(quantity.Int32),
				Price:       float64(price.Float32),
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("rows iteration: %w", err)
	}

	if !orderFound {
		return nil, nil, ErrNotFound
	}

	return order, items, nil

}

func (r *orderRepo) GetByCustomerID(ctx context.Context, customerID int) ([]models.Order, error) {
	if customerID <= 0 {
		return nil, fmt.Errorf("%w: ID must be positive", ErrInvalidInput)
	}

	sql := `SELECT 
		order_id,
		customer_id,
		total_amount,
		status,
		created_at
		FROM orders
		WHERE customer_id = $1`

	rows, err := r.db.Query(ctx, sql, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders by customerID %d: %w", customerID, err)
	}

	defer rows.Close()

	var orders []models.Order

	for rows.Next() {
		var o models.Order

		err := rows.Scan(&o.OrderID,
			&o.CustomerID,
			&o.TotalAmount,
			&o.Status,
			&o.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan orders by customerID: %w", err)
		}

		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to complete row iteration: %w", err)
	}

	return orders, nil
}
