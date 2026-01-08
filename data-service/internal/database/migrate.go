package database

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

func Migrate(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations: %w", err)
	}

	files, err := os.ReadDir("./internal/database/migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	var upMigrations []string

	fmt.Printf("Found %d files:\n", len(files))
	for _, file := range files {
		name := file.Name()

		if strings.HasSuffix(name, ".up.sql") {
			upMigrations = append(upMigrations, name)
		}
	}

	sort.Strings(upMigrations)

	fmt.Printf("Found %d migrations:\n", len(upMigrations))

	query := "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)"

	for _, migration := range upMigrations {

		var exists bool

		// проверяем выполнена ли миграция

		err := conn.QueryRow(context.Background(), query, migration).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check migrations %s: %w:", migration, err)
		}

		if exists {
			fmt.Println("migration is already applied, \n:", migration)
			continue
		}

		// читаем миграции (sql файлы)

		filePath := ("./internal/database/migrations/" + migration)

		sqlBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read sql file: %s: %w", migration, err)
		}

		sql := string(sqlBytes)

		// выполняем миграции (sql файлы)

		_, err = conn.Exec(context.Background(), sql)
		if err != nil {
			return fmt.Errorf("failed to complete sql file: %s: %w", migration, err)
		}

		// записываем в schema_migrator

		insertQuery := "INSERT INTO schema_migrations (version) VALUES ($1)"
		_, err = conn.Exec(context.Background(),
			insertQuery,
			migration,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration, err)
		}
	}

	return nil
}
