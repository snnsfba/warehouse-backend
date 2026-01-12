package repository

import (
	"context"
	"data-service/internal/models"
)

type ProductRepository interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id int) (*models.Product, error)
	GetAll(ctx context.Context) ([]models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, id int) error

	UpdateQuantity(ctx context.Context, id int, change int) error
	GetByCategory(ctx context.Context, category string) ([]models.Product, error)
}

type CustomerRepository interface {
	Create(ctx context.Context, customer *models.Customer) error
	GetByID(ctx context.Context, id int) (*models.Customer, error)
	GetAll(ctx context.Context) ([]models.Customer, error)
	Update(ctx context.Context, customer *models.Customer) error
	Delete(ctx context.Context, id int) error

	GetByEmail(ctx context.Context, email string) (*models.Customer, error)
	GetByPhoneNumber(ctx context.Context, phoneNumber string) (*models.Customer, error)
}
