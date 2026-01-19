package repository

import (
	"context"
	"data-service/internal/models"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type operationRepo struct {
	db *pgx.Conn
}

func NewOperationRepository(db *pgx.Conn) OperationRepository {
	return &operationRepo{db: db}
}

func (r *operationRepo) Create(ctx context.Context, o *models.Operation) error {
	var orderID interface{}
	if o.OrderID != nil && *o.OrderID > 0 {
		orderID = *o.OrderID
	} else {
		orderID = nil
	}
	if o == nil {
		return fmt.Errorf("%w: operation cannot be nil", ErrInvalidInput)
	}
	if o.ProductID <= 0 {
		return fmt.Errorf("%w: product ID must be positive", ErrInvalidInput)
	}
	if o.ChangeQuant == 0 {
		return fmt.Errorf("%w: the variable quantity cannot be 0", ErrInvalidInput)
	}
	validStatuses := map[string]bool{
		"incoming":   true,
		"outgoing":   true,
		"adjustment": true,
		"reserve":    true,
	}
	if !validStatuses[o.OperationType] {
		return fmt.Errorf("%w: invalid status '%s'", ErrInvalidInput, o.OperationType)
	}

	sql := ` INSERT INTO operations (
		product_id,
		order_id,
		operation_type,
		change_quant,
		created_at
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING operation_id
	`

	now := time.Now()
	o.CreatedAt = now

	err := r.db.QueryRow(ctx, sql,
		o.ProductID,
		orderID,
		o.OperationType,
		o.ChangeQuant,
		o.CreatedAt,
	).Scan(&o.OperationID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("failed to create operation: %w", err)
	}
	return nil
}

func (r *operationRepo) GetByProductID(ctx context.Context, productID int) ([]models.Operation, error) {
	if productID <= 0 {
		return nil, fmt.Errorf("%w: ID must be positive", ErrInvalidInput)
	}

	sql := `SELECT 
		product_id,
		order_id,
		operation_type,
		change_quant,
		created_at
		FROM operations
		WHERE product_id = $1
		`
	rows, err := r.db.Query(ctx, sql, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations by product ID %d: %w", productID, err)
	}

	defer rows.Close()

	var operations []models.Operation

	for rows.Next() {
		var o models.Operation

		err := rows.Scan(&o.ProductID,
			&o.OrderID,
			&o.OperationType,
			&o.ChangeQuant,
			&o.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan operations by product ID: %w", err)
		}

		operations = append(operations, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to complete row iteration: %w", err)
	}

	return operations, nil
}

func (r *operationRepo) GetByOrderID(ctx context.Context, orderID int) ([]models.Operation, error) {
	if orderID <= 0 {
		return nil, fmt.Errorf("%w: ID must be positive", ErrInvalidInput)
	}

	sql := ` SELECT
		product_id,
		order_id,
		operation_type,
		change_quant,
		created_at
		FROM operations
		WHERE order_id = $1
		`

	rows, err := r.db.Query(ctx, sql, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations by order ID: %w", err)
	}

	defer rows.Close()

	var operations []models.Operation

	for rows.Next() {
		var o models.Operation

		err := rows.Scan(&o.ProductID,
			&o.OrderID,
			&o.OperationType,
			&o.ChangeQuant,
			&o.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan operations by order ID: %w", err)
		}

		operations = append(operations, o)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to complete rows iteration: %w", err)
	}

	return operations, nil
}
