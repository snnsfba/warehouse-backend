package repository

import (
	"context"
	"data-service/internal/models"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type productRepo struct {
	db *pgx.Conn
}

func NewProductRepository(db *pgx.Conn) ProductRepository {
	return &productRepo{db: db}
}

func (r *productRepo) Create(ctx context.Context, p *models.Product) error {
	if p.Name == "" {
		return fmt.Errorf("%w: product name required", ErrInvalidInput)
	}
	if p.Price <= 0 {
		return fmt.Errorf("%w: product price  should be positive", ErrInvalidInput)
	}
	if p.Quantity < 0 {
		return fmt.Errorf("%w: product quantity cannot be negative", ErrInvalidInput)
	}

	sql := `
		INSERT INTO products (
			name,
			price,
			description,
			quantity,
			category,
			created_at,
			updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING product_id
	`

	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now

	err := r.db.QueryRow(ctx, sql,
		p.Name,
		p.Price,
		p.Description,
		p.Quantity,
		p.Category,
		p.CreatedAt,
		p.UpdatedAt,
	).Scan(&p.ProductID)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

func (r *productRepo) GetByID(ctx context.Context, id int) (*models.Product, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: ID cannot be empty", ErrInvalidInput)
	}

	sql := `
		SELECT 
			product_id,
			name,
			price,
			description,
			quantity,
			category,
			created_at,
			updated_at	
		FROM products WHERE product_id = $1
		`

	var product models.Product

	err := r.db.QueryRow(ctx, sql, id).Scan(
		&product.ProductID,
		&product.Name,
		&product.Price,
		&product.Description,
		&product.Quantity,
		&product.Category,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get product by id %d: %w", id, err)
	}

	return &product, nil

}

func (r *productRepo) GetAll(ctx context.Context) ([]models.Product, error) {
	sql := `
    SELECT 
        product_id,
        name,
		price,
		description,
		quantity,
		category,
		created_at,
		updated_at	
    FROM products 
    ORDER BY product_id
`
	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to get all products: %w", err)
	}

	defer rows.Close()

	var products []models.Product

	for rows.Next() {
		var p models.Product

		err := rows.Scan(&p.ProductID,
			&p.Name,
			&p.Price,
			&p.Description,
			&p.Quantity,
			&p.Category,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan products: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to complete row iteration: %w", err)
	}

	return products, nil
}

func (r *productRepo) Update(ctx context.Context, p *models.Product) error {
	if p.Name == "" {
		return fmt.Errorf("%w: product name required", ErrInvalidInput)
	}
	if p.Price <= 0 {
		return fmt.Errorf("%w: product price  should be positive", ErrInvalidInput)
	}
	if p.Quantity < 0 {
		return fmt.Errorf("%w: product quantity cannot be negative", ErrInvalidInput)
	}
	if p.ProductID <= 0 {
		return fmt.Errorf("%w: ID cannot be empty", ErrInvalidInput)
	}

	sql := `
	UPDATE products 
	SET 
		name = $1,
		price = $2,
    	description = $3,
    	quantity = $4,
    	category = $5,
		updated_at = $6
	WHERE product_id = $7
	RETURNING updated_at
	`

	now := time.Now()

	err := r.db.QueryRow(ctx, sql,
		p.Name,
		p.Price,
		p.Description,
		p.Quantity,
		p.Category,
		now,
		p.ProductID,
	).Scan(&p.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("failed to update product %d: %w", p.ProductID, err)
	}

	return nil
}

func (r *productRepo) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("%w: ID cannot be empty", ErrInvalidInput)
	}

	sql := `DELETE FROM products WHERE product_id = $1`

	result, err := r.db.Exec(ctx, sql, id)
	if err != nil {
		return fmt.Errorf("failed to delete product %d: %w", id, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil

}

func (r *productRepo) UpdateQuantity(ctx context.Context, id int, change int) error {
	product, err := r.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}

	newQuantity := product.Quantity + change
	if newQuantity < 0 {
		return fmt.Errorf("%w: insufficient quantity. Current: %d, Requested change: %d", ErrNotEnough, product.Quantity, change)
	}

	sql := `UPDATE products SET 
	quantity = quantity + $1,
		updated_at = $2
	WHERE product_id = $3
	RETURNING quantity, updated_at
	`

	now := time.Now()
	var returnedQuantity int
	var returnedUpdatedAt time.Time

	err = r.db.QueryRow(ctx, sql, change, now, id).Scan(&returnedQuantity, &returnedUpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("failed to update product quantity %d: %w", id, err)
	}

	if returnedQuantity != newQuantity {
		return fmt.Errorf("quantity mismatch after update: expected %d, got %d",
			newQuantity, returnedQuantity)
	}

	return nil
}

func (r *productRepo) GetByCategory(ctx context.Context, category string) ([]models.Product, error) {
	if category == "" {
		return nil, fmt.Errorf(" category cannot be empty: %w", ErrInvalidInput)
	}

	sql := `
		SELECT 
			product_id,
			name,
			price,
			description,
			quantity,
			category,
			created_at,
			updated_at	
		FROM products WHERE category = $1
		ORDER BY product_id
		`

	rows, err := r.db.Query(ctx, sql, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get products with category: %w", err)
	}

	defer rows.Close()

	var products []models.Product

	for rows.Next() {
		var p models.Product

		err := rows.Scan(&p.ProductID,
			&p.Name,
			&p.Price,
			&p.Description,
			&p.Quantity,
			&p.Category,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan products: %w", err)
		}
		products = append(products, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to complete row iteration: %w", err)
	}

	return products, nil

}
