package repository

import "github.com/jackc/pgx/v5"

type operationRepo struct {
	db *pgx.Conn
}

func NewOperationRepository