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

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *models.Order, items []models.OrderItem) error
	GetByID(ctx context.Context, id int) (*models.Order, error)
	GetAll(ctx context.Context) ([]models.Order, error)
	UpdateStatus(ctx context.Context, id int, status string) error

	GetByCustomerID(ctx context.Context, customerID int) ([]models.Order, error)
	GetOrderWithItems(ctx context.Context, id int) (*models.Order, []models.OrderItem, error)
}

type OperationRepository interface {
	Create(ctx context.Context, operation *models.Operation) error
	GetByProductID(ctx context.Context, productID int) ([]models.Operation, error)
	GetByOrderID(ctx context.Context, orderID int) ([]models.Operation, error)
}
