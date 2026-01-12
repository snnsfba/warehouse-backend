package repository

import (
	"context"
	"data-service/internal/models"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type customerRepo struct {
	db *pgx.Conn
}

var validate = validator.New()

func NewCustomerRepository(db *pgx.Conn) CustomerRepository {
	return &customerRepo{db: db}
}

func (r *customerRepo) Create(ctx context.Context, c *models.Customer) error {
	if err := validate.Struct(c); err != nil {
		var validationErr validator.ValidationErrors
		if errors.As(err, &validationErr) {
			firstErr := validationErr[0]
			switch firstErr.Field() {
			case "Email":
				return fmt.Errorf("%w: invalid email format", ErrInvalidInput)
			case "PhoneNumber":
				return fmt.Errorf("%w: phone_number must be in E.164 format (+79161234567)", ErrInvalidInput)

			case "Name":
				return fmt.Errorf("%w: name must be 2-150 characters", ErrInvalidInput)

			}
		}
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	sql := `
		INSERT INTO customers (
			name,
			phone_number,
			address,
			email,
			registered_at
	) VALUES ($1, $2, $3, $4, $5)
	RETURNING customer_id
	`

	now := time.Now()
	c.RegisteredAt = now

	err := r.db.QueryRow(ctx, sql,
		c.Name,
		c.PhoneNumber,
		c.Address,

		c.Email,
		c.RegisteredAt,
	).Scan(&c.CustomerID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "email") {
				return fmt.Errorf("%w: email already exists", ErrDuplicate)
			}
			if strings.Contains(pgErr.ConstraintName, "customers_phone_number_key") {
				return fmt.Errorf("%w: phone_number already exists", ErrDuplicate)
			}
		}
		return fmt.Errorf("create customer: %w", err)
	}

	return nil

}

func (r *customerRepo) GetByID(ctx context.Context, id int) (*models.Customer, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: ID cannot be empty", ErrInvalidInput)
	}

	sql := `
		SELECT
		customer_id,
		name,
		phone_number,
		address,
		email,
		registered_at
		FROM customers WHERE customer_id = $1
	`

	var customer models.Customer

	err := r.db.QueryRow(ctx, sql, id).Scan(
		&customer.CustomerID,
		&customer.Name,
		&customer.PhoneNumber,
		&customer.Address,
		&customer.Email,
		&customer.RegisteredAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get customer with id %d: %w", id, err)
	}

	return &customer, nil
}

func (r *customerRepo) GetAll(ctx context.Context) ([]models.Customer, error) {
	sql := `
	SELECT
	customer_id,
	name,
	phone_number,
	address,
	email,
	registered_at
	FROM customers
	ORDER BY customer_id`

	rows, err := r.db.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("failed to get all customers: %w", err)
	}

	defer rows.Close()

	var customers []models.Customer

	for rows.Next() {
		var c models.Customer

		err := rows.Scan(&c.CustomerID,
			&c.Name,
			&c.PhoneNumber,
			&c.Address,
			&c.Email,
			&c.RegisteredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customers: %w", err)
		}
		customers = append(customers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to complete row iteration: %w", err)
	}

	return customers, nil
}

func (r *customerRepo) Update(ctx context.Context, c *models.Customer) error {
	return nil
}

func (r *customerRepo) Delete(ctx context.Context, id int) error {
	return nil
}

func (r *customerRepo) GetByEmail(ctx context.Context, email string) (*models.Customer, error) {
	return nil, fmt.Errorf("method not implemented")
}

func (r *customerRepo) GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Customer, error) {
	return nil, fmt.Errorf("method not implemented")
}
