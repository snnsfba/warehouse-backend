package repository

import "errors"

var (
	ErrNotFound        = errors.New("resourсe not found")
	ErrDuplicate       = errors.New("duplicate resource")
	ErrInvalidInput    = errors.New("invalid input data")
	ErrNotEnough       = errors.New("not enough quantity available")
	ErrProductNotFound = errors.New("product not found")
	ErrCustomerExists  = errors.New("customer already exists")
)
