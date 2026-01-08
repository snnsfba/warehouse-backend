package main

import (
	"context"
	"data-service/internal/database"
	"fmt"
	"log"
	"time"
)

func main() {
	cfg, err := database.LoadConfig()
	if err != nil {
		log.Fatal("failed to load config", err)
	}

	conn, err := database.Connect(cfg)
	if err != nil {
		log.Fatal("failed to connect database", err)
	}
	defer conn.Close(context.Background())

	fmt.Println("Succesfully connect to DB!")

	var result time.Time

	fmt.Println("\n=== Starting migrations ===")
	err = database.Migrate(conn)
	if err != nil {
		log.Fatal("migrations failed:", err)
	}
	fmt.Println("=== MIgrations complete ===")
	err = conn.QueryRow(context.Background(), "SELECT NOW()").Scan(&result)
	if err != nil {
		log.Fatal("Query failed", err)
	}

	fmt.Println("Current database time:", result)
	fmt.Printf("User: '%s'\n", cfg.User)
	fmt.Printf("Password: '%s'\n", cfg.Password)

}
