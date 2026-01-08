package database

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	return &Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "app_user"),
		Password: getEnv("DB_PASSWORD", "postgres_password"),
		DBName:   getEnv("DB_NAME", "app_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}, nil

}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
