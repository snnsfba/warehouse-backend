package main

import (
	"context"
	"data-service/internal/database"
	"data-service/internal/models"
	"data-service/internal/repository"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
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

	var result time.Time

	err = database.Migrate(conn)
	if err != nil {
		log.Fatal("migrations failed:", err)
	}
	err = conn.QueryRow(context.Background(), "SELECT NOW()").Scan(&result)
	if err != nil {
		log.Fatal("Query failed", err)
	}

	fmt.Println("Current database time:", result)
	fmt.Printf("User: '%s'\n", cfg.User)
	fmt.Printf("Password: '%s'\n", cfg.Password)

	testCustomerRepository(conn)

}

func testCustomerRepository(conn *pgx.Conn) {
	repo := repository.NewCustomerRepository(conn)

	// //TECT 1: create

	customer1 := &models.Customer{
		Name:        "Серега",
		PhoneNumber: "+79778756911",
		Address:     "Russia, Moscow",
		Email:       "Vyacheslaavv@gmail.com",
	}

	err := repo.Create(context.Background(), customer1)
	if err != nil {
		log.Fatal("Create test failed", err)
		return
	}
	fmt.Printf("Created customer ID: %d\n", customer1.CustomerID)

	// // Тест 2: Дубликат email
	// customer2 := &models.Customer{
	// 	Name:        "Серега",
	// 	PhoneNumber: "+79161234567",
	// 	Address:     "Tojikistan, Dushanbe",
	// 	Email:       "Vyacheslav@gmail.com",
	// }

	// err = repo.Create(context.Background(), customer2)
	// if err == nil || !strings.Contains(err.Error(), "already exist") {
	// 	log.Fatal("Should reject duplicate email")
	// }
	// fmt.Printf("Correctly rejected duplicate email: %v\n", err)

	// // Тест 3: Дубликат phone
	// customer3 := &models.Customer{
	// 	Name:        "Санек",
	// 	PhoneNumber: "+79778756980",
	// 	Address:     "Uzbekistan, Mbappe",
	// 	Email:       "Pidor1488@gmail.com",
	// }

	// err = repo.Create(context.Background(), customer3)
	// if err == nil || !strings.Contains(err.Error(), "already exists") {
	// 	log.Fatal("Should reject duplicate phone")
	// }
	// fmt.Printf("Correctly rejected duplicate phone: %v\n", err)

	// //Тест 4: Невалидный email

	// customer4 := &models.Customer{
	// 	Name:        "Данек",
	// 	PhoneNumber: "+79882132233",
	// 	Email:       "Xyesos228",
	// 	Address:     "Zalupaches",
	// }

	// err = repo.Create(context.Background(), customer4)
	// if err == nil {
	// 	log.Fatal("should reject invalid Email", err)
	// }
	// fmt.Printf("Correctly rejected invalid Email: %v\n", err)
	//
	//
	//
	//
	// Тест 5: GetByID

	savedCustomer, err := repo.GetByID(context.Background(), customer1.CustomerID)
	if err != nil {
		log.Fatal("GetByID test failed:", err)
		return
	}
	fmt.Printf("Retrieved customer:\n")
	fmt.Printf("ID: %d\n", savedCustomer.CustomerID)
	fmt.Printf("Name: %s\n", savedCustomer.Name)

	// Тест 6: GetByID с несуществующим ID

	_, err = repo.GetByID(context.Background(), 99999)
	if err != nil {
		fmt.Printf("Correctly got error for non-existent product: %v\n", err)
	} else {
		fmt.Println("Should have gotten error for non-existent product")
	}
	//
	//
	//
	//

	//TECT 5: GetAll
	customer5 := &models.Customer{
		Name:        "Алексей",
		PhoneNumber: "+79161112233",
		Email:       "alexey@example.com",
	}
	repo.Create(context.Background(), customer5)

	allCustomers, err := repo.GetAll(context.Background())
	if err != nil {
		log.Fatal("GetAll failed:", err)
	}

	fmt.Printf("Found %d customers:\n", len(allCustomers))
	for i, c := range allCustomers {
		fmt.Printf("  %d. ID: %d, Name: %s, Email: %s\n",
			i+1, c.CustomerID, c.Name, c.Email)
	}

	// Проверяем что получили хотя бы 2 клиента
	if len(allCustomers) < 2 {
		log.Fatal("Should have at least 2 customers")
	}

}

// func testProductRepository(conn *pgx.Conn) {
// 	repo := repository.NewProductRepository(conn)

// 	// Тест 1: Create
// 	product := &models.Product{
// 		Name:     "Logitech MX Master 3",
// 		Quantity: 8,
// 		Price:    8999.90,
// 	}

// 	err := repo.Create(context.Background(), product)
// 	if err != nil {
// 		log.Println("Create test failed:", err)
// 		return
// 	}
// 	fmt.Printf("Created product ID: %d\n", product.ProductID)

// 	// Тест 2: GetByID

// 	savedProduct, err := repo.GetByID(context.Background(), product.ProductID)
// 	if err != nil {
// 		log.Println("GetByID test failed:", err)
// 		return
// 	}

// 	fmt.Printf("Retrieved product:\n")
// 	fmt.Printf("ID: %d\n", savedProduct.ProductID)
// 	fmt.Printf("Name: %s\n", savedProduct.Name)
// 	fmt.Printf("Quantity: %d\n", savedProduct.Quantity)
// 	fmt.Printf("Price: %.2f\n", savedProduct.Price)

// 	// Тест 3: GetByID с несуществующим ID
// 	fmt.Println("\n=== Testing GetByID with invalid ID ===")
// 	_, err = repo.GetByID(context.Background(), 99999)
// 	if err != nil {
// 		fmt.Printf("Correctly got error for non-existent product: %v\n", err)
// 	} else {
// 		fmt.Println("Should have gotten error for non-existent product")
// 	}

// 	// Тест 4: GetAll

// 	fmt.Println("\n=== Testing GetAll  ===")

// 	repo.Create(context.Background(), &models.Product{
// 		Name:     "DJI Mavic 3Pro",
// 		Quantity: 120,
// 		Price:    29999.99,
// 	})

// 	repo.Create(context.Background(), &models.Product{
// 		Name:     "Монитор 27\"",
// 		Quantity: 5,
// 		Price:    21999.99,
// 	})

// 	allProducts, err := repo.GetAll(context.Background())
// 	if err != nil {
// 		log.Println("GetAll failed:", err)
// 		return
// 	}

// 	fmt.Printf("Found %d products:\n", len(allProducts))

// 	for i, p := range allProducts {
// 		fmt.Printf("  %d. ID: %d, Name: %s, Qty: %d, Price: %.2f\n",
// 			i+1, p.ProductID, p.Name, p.Quantity, p.Price)
// 	}

// 	// Тест 5: Update
// 	fmt.Println("\n=== Testing Update ===")

// 	product, _ = repo.GetByID(context.Background(), 2) // ID из предыдущего теста

// 	oldName := product.Name
// 	product.Name = "UPDATED: " + product.Name
// 	product.Price = product.Price + 1000

// 	err = repo.Update(context.Background(), product)
// 	if err != nil {
// 		log.Fatal("Update failed:", err)
// 	}
// 	fmt.Println("Product updated")

// 	updated, _ := repo.GetByID(context.Background(), product.ProductID)
// 	if updated.Name == oldName {
// 		log.Fatal("Name didn't change!")
// 	}
// 	fmt.Printf("   Old name: %s\n", oldName)
// 	fmt.Printf("   New name: %s\n", updated.Name)

// 	// Тест 6: Delete
// 	fmt.Println("\n=== Testing Delete ===")

// 	toDelete := &models.Product{
// 		Name:     "Товар для удаления",
// 		Quantity: 5,
// 		Price:    1000,
// 	}
// 	repo.Create(context.Background(), toDelete)
// 	fmt.Printf("Created product ID %d for deletion test\n", toDelete.ProductID)

// 	err = repo.Delete(context.Background(), toDelete.ProductID)
// 	if err != nil {
// 		log.Fatal("Delete failed:", err)
// 	}
// 	fmt.Println("Product deleted")

// 	_, err = repo.GetByID(context.Background(), toDelete.ProductID)
// 	if err != repository.ErrNotFound {
// 		log.Fatal("Deleted product should not be found")
// 	}
// 	fmt.Println("GetByID correctly returns ErrNotFound")

// 	err = repo.Delete(context.Background(), 99999)
// 	if err != repository.ErrNotFound {
// 		log.Fatal("Delete should return ErrNotFound for non-existent product")
// 	}
// 	fmt.Println("Delete returns ErrNotFound for non-existent product")

// 	err = repo.Delete(context.Background(), 0)
// 	if !errors.Is(err, repository.ErrInvalidInput) {
// 		log.Fatal("Should return ErrInvalidInput for ID=0")
// 	}
// 	fmt.Println("Delete validates ID correctly")

// 	// Тест 7: UpdateQuant
// 	fmt.Println("\n=== Testing UpdateQuantity ===")

// 	testProduct := &models.Product{
// 		Name:     "Тест UpdateQuantity",
// 		Quantity: 10,
// 		Price:    1000,
// 	}
// 	repo.Create(context.Background(), testProduct)

// 	repo.UpdateQuantity(context.Background(), testProduct.ProductID, 5)
// 	updated, _ = repo.GetByID(context.Background(), testProduct.ProductID)
// 	fmt.Printf("+5: %d → %d\n", 10, updated.Quantity)

// 	repo.UpdateQuantity(context.Background(), testProduct.ProductID, -3)
// 	updated, _ = repo.GetByID(context.Background(), testProduct.ProductID)
// 	fmt.Printf("-3: %d → %d\n", 15, updated.Quantity)

// 	err = repo.UpdateQuantity(context.Background(), testProduct.ProductID, -20)
// 	if err != nil {
// 		fmt.Printf("Correctly rejected: %v\n", err)
// 	}

// 	fmt.Println("\n=== Testing GetByCategory ===")

// 	// Создаем товары с разными категориями
// 	repo.Create(context.Background(), &models.Product{
// 		Name:     "Ноутбук",
// 		Category: "Электроника",
// 		Quantity: 5,
// 		Price:    50000,
// 	})

// 	repo.Create(context.Background(), &models.Product{
// 		Name:     "Монитор",
// 		Category: "Электроника",
// 		Quantity: 3,
// 		Price:    20000,
// 	})

// 	repo.Create(context.Background(), &models.Product{
// 		Name:     "Стул",
// 		Category: "Мебель",
// 		Quantity: 10,
// 		Price:    5000,
// 	})

// 	// Тест 1: Категория с товарами
// 	electronicProducts, err := repo.GetByCategory(context.Background(), "Электроника")
// 	if err != nil {
// 		log.Fatal("❌ GetByCategory failed:", err)
// 	}
// 	fmt.Printf("✅ Электроника: %d товаров\n", len(electronicProducts))
// 	for _, p := range electronicProducts {
// 		fmt.Printf("   - %s (%.2f руб)\n", p.Name, p.Price)
// 	}

// 	// Тест 2: Категория без товаров
// 	nonexistent, err := repo.GetByCategory(context.Background(), "Одежда")
// 	if err != nil {
// 		log.Fatal("❌ GetByCategory should not error for empty category")
// 	}
// 	fmt.Printf("✅ Одежда: %d товаров (пустая категория ок)\n", len(nonexistent))

// }
