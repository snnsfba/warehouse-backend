package main

import (
	"context"
	"data-service/internal/database"
	"data-service/internal/models"
	"data-service/internal/repository"
	"errors"
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

	testOrderRepository(conn)

}

func testOrderRepository(conn *pgx.Conn) {
	fmt.Println("\n=== Testing OrderRepository ===")

	repo := repository.NewOrderRepository(conn)

	// Сначала создаем тестовые данные
	fmt.Println("Setting up test data...")

	// 1. Создаем тестового клиента
	customerRepo := repository.NewCustomerRepository(conn)
	testCustomer := &models.Customer{
		Name:        "Тестовый Клиент",
		Email:       "test-order@example.com",
		PhoneNumber: "+79169998877",
	}
	customerRepo.Create(context.Background(), testCustomer)

	// 2. Создаем тестовые заказы напрямую через SQL
	var orderIDs []int
	for i := 1; i <= 3; i++ {
		var orderID int
		amount := float64(1000 * i)
		conn.QueryRow(context.Background(),
			`INSERT INTO orders (customer_id, total_amount, status)
             VALUES ($1, $2, $3) RETURNING order_id`,
			testCustomer.CustomerID, amount, "created",
		).Scan(&orderID)
		orderIDs = append(orderIDs, orderID)
	}

	fmt.Printf("✅ Created test customer ID: %d\n", testCustomer.CustomerID)
	fmt.Printf("✅ Created test orders IDs: %v\n", orderIDs)

	// === ТЕСТ 1: GetByID ===
	fmt.Println("\n=== Testing GetByID ===")

	// 1.1 Существующий заказ
	order, err := repo.GetByID(context.Background(), orderIDs[0])
	if err != nil {
		log.Fatal("❌ GetByID failed:", err)
	}
	fmt.Printf("✅ GetByID found order: ID %d, Status: %s, Amount: %.2f\n",
		order.OrderID, order.Status, order.TotalAmount)

	// 1.2 Несуществующий заказ
	_, err = repo.GetByID(context.Background(), 99999)
	if err != repository.ErrNotFound {
		log.Fatal("❌ GetByID should return ErrNotFound for non-existent order")
	}
	fmt.Println("✅ GetByID correctly returns ErrNotFound")

	// 1.3 Невалидный ID
	_, err = repo.GetByID(context.Background(), 0)
	if !errors.Is(err, repository.ErrInvalidInput) {
		log.Fatal("❌ GetByID should validate ID")
	}
	fmt.Println("✅ GetByID validates ID correctly")

	// === ТЕСТ 2: GetAll ===
	fmt.Println("\n=== Testing GetAll ===")

	allOrders, err := repo.GetAll(context.Background())
	if err != nil {
		log.Fatal("❌ GetAll failed:", err)
	}
	fmt.Printf("✅ GetAll found %d orders\n", len(allOrders))

	if len(allOrders) < 3 {
		log.Fatal("❌ Should have at least 3 orders")
	}

	// Проверяем порядок (ORDER BY order_id)
	for i := 0; i < len(allOrders)-1; i++ {
		if allOrders[i].OrderID > allOrders[i+1].OrderID {
			log.Fatal("❌ Orders should be sorted by order_id")
		}
	}
	fmt.Println("✅ Orders are correctly sorted")

	// === ТЕСТ 3: UpdateStatus ===
	fmt.Println("\n=== Testing UpdateStatus ===")

	// 3.1 Успешное обновление
	err = repo.UpdateStatus(context.Background(), orderIDs[0], "paid")
	if err != nil {
		log.Fatal("❌ UpdateStatus failed:", err)
	}

	// Проверяем через GetByID
	updatedOrder, _ := repo.GetByID(context.Background(), orderIDs[0])
	if updatedOrder.Status != "paid" {
		log.Fatal("❌ Status didn't change to 'paid'")
	}
	fmt.Println("✅ UpdateStatus changed status to 'paid'")

	// 3.2 Несуществующий заказ
	err = repo.UpdateStatus(context.Background(), 99999, "paid")
	if err != repository.ErrNotFound {
		log.Fatal("❌ UpdateStatus should return ErrNotFound")
	}
	fmt.Println("✅ UpdateStatus returns ErrNotFound correctly")

	// 3.3 Невалидный статус
	err = repo.UpdateStatus(context.Background(), orderIDs[1], "invalid_status")
	if !errors.Is(err, repository.ErrInvalidInput) {
		log.Fatal("❌ Should reject invalid status")
	}
	fmt.Println("✅ UpdateStatus validates status correctly")

	// 3.4 Пустой статус
	err = repo.UpdateStatus(context.Background(), orderIDs[1], "")
	if !errors.Is(err, repository.ErrInvalidInput) {
		log.Fatal("❌ Should reject empty status")
	}
	fmt.Println("✅ UpdateStatus rejects empty status")

	// === ТЕСТ 4: GetByCustomerID ===
	fmt.Println("\n=== Testing GetByCustomerID ===")

	// 4.1 Заказы существующего клиента
	customerOrders, err := repo.GetByCustomerID(context.Background(), testCustomer.CustomerID)
	if err != nil {
		log.Fatal("❌ GetByCustomerID failed:", err)
	}
	fmt.Printf("✅ Found %d orders for customer ID %d\n",
		len(customerOrders), testCustomer.CustomerID)

	if len(customerOrders) < 3 {
		log.Fatal("❌ Should have 3 orders for test customer")
	}

	// Проверяем что все заказы этого клиента
	for _, o := range customerOrders {
		if o.CustomerID != testCustomer.CustomerID {
			log.Fatal("❌ GetByCustomerID returned wrong customer's order")
		}
	}
	fmt.Println("✅ All orders belong to correct customer")

	// 4.2 Клиент без заказов (создадим нового)
	newCustomer := &models.Customer{
		Name:        "Беззаказный",
		Email:       "no-orders@example.com",
		PhoneNumber: "+79161112233",
	}
	customerRepo.Create(context.Background(), newCustomer)

	emptyOrders, err := repo.GetByCustomerID(context.Background(), newCustomer.CustomerID)
	if err != nil {
		log.Fatal("❌ GetByCustomerID should work for customers without orders")
	}
	if len(emptyOrders) != 0 {
		log.Fatal("❌ Should return empty slice for customer without orders")
	}
	fmt.Println("✅ GetByCustomerID returns empty slice correctly")

	// 4.3 Невалидный customerID
	_, err = repo.GetByCustomerID(context.Background(), 0)
	if !errors.Is(err, repository.ErrInvalidInput) {
		log.Fatal("❌ Should validate customerID")
	}
	fmt.Println("✅ GetByCustomerID validates customerID")

	fmt.Println("\n=== Testing GetOrderWithItems ===")

	// 1. Создаем тестовый продукт
	productRepo := repository.NewProductRepository(conn)
	testProduct := &models.Product{
		Name:     "Тестовый товар для заказа",
		Quantity: 100,
		Price:    1500.00,
		Category: "Тест",
	}
	productRepo.Create(context.Background(), testProduct)

	// 2. Создаем заказ с товарами напрямую через SQL
	var orderID int
	conn.QueryRow(context.Background(),
		`INSERT INTO orders (customer_id, total_amount, status)
     VALUES ($1, $2, $3) RETURNING order_id`,
		1, 5000.00, "created",
	).Scan(&orderID)

	// 3. Добавляем товары в заказ
	conn.Exec(context.Background(),
		`INSERT INTO order_items (order_id, product_id, quantity, price)
     VALUES ($1, $2, $3, $4)`,
		orderID, testProduct.ProductID, 2, 1500.00,
	)

	conn.Exec(context.Background(),
		`INSERT INTO order_items (order_id, product_id, quantity, price)
     VALUES ($1, $2, $3, $4)`,
		orderID, testProduct.ProductID, 1, 2000.00,
	)

	fmt.Printf("✅ Created order with items: ID %d\n", orderID)

	// 4. Тестируем GetOrderWithItems
	order, items, err := repo.GetOrderWithItems(context.Background(), orderID)
	if err != nil {
		log.Fatal("❌ GetOrderWithItems failed:", err)
	}

	fmt.Printf("✅ Found order: ID %d, Status: %s\n", order.OrderID, order.Status)
	fmt.Printf("✅ Found %d items in order:\n", len(items))
	for i, item := range items {
		fmt.Printf("   %d. ProductID: %d, Quantity: %d, Price: %.2f\n",
			i+1, item.ProductID, item.Quantity, item.Price)
	}

	// 5. Тест: заказ без товаров (создаем новый заказ)
	var emptyOrderID int
	conn.QueryRow(context.Background(),
		`INSERT INTO orders (customer_id, total_amount, status)
     VALUES ($1, $2, $3) RETURNING order_id`,
		1, 1000.00, "created",
	).Scan(&emptyOrderID)

	_, emptyItems, err := repo.GetOrderWithItems(context.Background(), emptyOrderID)
	if err != nil {
		log.Fatal("❌ GetOrderWithItems for empty order failed:", err)
	}
	if len(emptyItems) != 0 {
		log.Fatal("❌ Empty order should have 0 items")
	}
	fmt.Println("✅ Empty order correctly returns 0 items")

	// 6. Тест: несуществующий заказ
	_, _, err = repo.GetOrderWithItems(context.Background(), 99999)
	if err != repository.ErrNotFound {
		log.Fatal("❌ Should return ErrNotFound for non-existent order")
	}
	fmt.Println("✅ Correctly returns ErrNotFound")

	fmt.Println("\n🎉 GetOrderWithItems TESTS PASSED!")
}

// 	fmt.Println("\n=== Testing OrderRepository ===")

// 	repo := repository.NewOrderRepository(conn)

// 	// Сначала создаем тестовые данные
// 	fmt.Println("Setting up test data...")

// 	// 1. Создаем тестового клиента
// 	customerRepo := repository.NewCustomerRepository(conn)
// 	testCustomer := &models.Customer{
// 		Name:        "Тестовый Клиент",
// 		Email:       "test-order@example.com",
// 		PhoneNumber: "+79169998877",
// 	}
// 	customerRepo.Create(context.Background(), testCustomer)

// 	// 2. Создаем тестовые заказы напрямую через SQL
// 	var orderIDs []int
// 	for i := 1; i <= 3; i++ {
// 		var orderID int
// 		amount := float64(1000 * i)
// 		conn.QueryRow(context.Background(),
// 			`INSERT INTO orders (customer_id, total_amount, status)
//              VALUES ($1, $2, $3) RETURNING order_id`,
// 			testCustomer.CustomerID, amount, "created",
// 		).Scan(&orderID)
// 		orderIDs = append(orderIDs, orderID)
// 	}

// 	fmt.Printf("✅ Created test customer ID: %d\n", testCustomer.CustomerID)
// 	fmt.Printf("✅ Created test orders IDs: %v\n", orderIDs)

// 	// === ТЕСТ 1: GetByID ===
// 	fmt.Println("\n=== Testing GetByID ===")

// 	// 1.1 Существующий заказ
// 	order, err := repo.GetByID(context.Background(), orderIDs[0])
// 	if err != nil {
// 		log.Fatal("❌ GetByID failed:", err)
// 	}
// 	fmt.Printf("✅ GetByID found order: ID %d, Status: %s, Amount: %.2f\n",
// 		order.OrderID, order.Status, order.TotalAmount)

// 	// 1.2 Несуществующий заказ
// 	_, err = repo.GetByID(context.Background(), 99999)
// 	if err != repository.ErrNotFound {
// 		log.Fatal("❌ GetByID should return ErrNotFound for non-existent order")
// 	}
// 	fmt.Println("✅ GetByID correctly returns ErrNotFound")

// 	// 1.3 Невалидный ID
// 	_, err = repo.GetByID(context.Background(), 0)
// 	if !errors.Is(err, repository.ErrInvalidInput) {
// 		log.Fatal("❌ GetByID should validate ID")
// 	}
// 	fmt.Println("✅ GetByID validates ID correctly")

// 	// === ТЕСТ 2: GetAll ===
// 	fmt.Println("\n=== Testing GetAll ===")

// 	allOrders, err := repo.GetAll(context.Background())
// 	if err != nil {
// 		log.Fatal("❌ GetAll failed:", err)
// 	}
// 	fmt.Printf("✅ GetAll found %d orders\n", len(allOrders))

// 	if len(allOrders) < 3 {
// 		log.Fatal("❌ Should have at least 3 orders")
// 	}

// 	// Проверяем порядок (ORDER BY order_id)
// 	for i := 0; i < len(allOrders)-1; i++ {
// 		if allOrders[i].OrderID > allOrders[i+1].OrderID {
// 			log.Fatal("❌ Orders should be sorted by order_id")
// 		}
// 	}
// 	fmt.Println("✅ Orders are correctly sorted")

// 	// === ТЕСТ 3: UpdateStatus ===
// 	fmt.Println("\n=== Testing UpdateStatus ===")

// 	// 3.1 Успешное обновление
// 	err = repo.UpdateStatus(context.Background(), orderIDs[0], "paid")
// 	if err != nil {
// 		log.Fatal("❌ UpdateStatus failed:", err)
// 	}

// 	// Проверяем через GetByID
// 	updatedOrder, _ := repo.GetByID(context.Background(), orderIDs[0])
// 	if updatedOrder.Status != "paid" {
// 		log.Fatal("❌ Status didn't change to 'paid'")
// 	}
// 	fmt.Println("✅ UpdateStatus changed status to 'paid'")

// 	// 3.2 Несуществующий заказ
// 	err = repo.UpdateStatus(context.Background(), 99999, "paid")
// 	if err != repository.ErrNotFound {
// 		log.Fatal("❌ UpdateStatus should return ErrNotFound")
// 	}
// 	fmt.Println("✅ UpdateStatus returns ErrNotFound correctly")

// 	// 3.3 Невалидный статус
// 	err = repo.UpdateStatus(context.Background(), orderIDs[1], "invalid_status")
// 	if !errors.Is(err, repository.ErrInvalidInput) {
// 		log.Fatal("❌ Should reject invalid status")
// 	}
// 	fmt.Println("✅ UpdateStatus validates status correctly")

// 	// 3.4 Пустой статус
// 	err = repo.UpdateStatus(context.Background(), orderIDs[1], "")
// 	if !errors.Is(err, repository.ErrInvalidInput) {
// 		log.Fatal("❌ Should reject empty status")
// 	}
// 	fmt.Println("✅ UpdateStatus rejects empty status")

// 	// === ТЕСТ 4: GetByCustomerID ===
// 	fmt.Println("\n=== Testing GetByCustomerID ===")

// 	// 4.1 Заказы существующего клиента
// 	customerOrders, err := repo.GetByCustomerID(context.Background(), testCustomer.CustomerID)
// 	if err != nil {
// 		log.Fatal("❌ GetByCustomerID failed:", err)
// 	}
// 	fmt.Printf("✅ Found %d orders for customer ID %d\n",
// 		len(customerOrders), testCustomer.CustomerID)

// 	if len(customerOrders) < 3 {
// 		log.Fatal("❌ Should have 3 orders for test customer")
// 	}

// 	// Проверяем что все заказы этого клиента
// 	for _, o := range customerOrders {
// 		if o.CustomerID != testCustomer.CustomerID {
// 			log.Fatal("❌ GetByCustomerID returned wrong customer's order")
// 		}
// 	}
// 	fmt.Println("✅ All orders belong to correct customer")

// 	// 4.2 Клиент без заказов (создадим нового)
// 	newCustomer := &models.Customer{
// 		Name:        "Беззаказный",
// 		Email:       "no-orders@example.com",
// 		PhoneNumber: "+79161112233",
// 	}
// 	customerRepo.Create(context.Background(), newCustomer)

// 	emptyOrders, err := repo.GetByCustomerID(context.Background(), newCustomer.CustomerID)
// 	if err != nil {
// 		log.Fatal("❌ GetByCustomerID should work for customers without orders")
// 	}
// 	if len(emptyOrders) != 0 {
// 		log.Fatal("❌ Should return empty slice for customer without orders")
// 	}
// 	fmt.Println("✅ GetByCustomerID returns empty slice correctly")

// 	// 4.3 Невалидный customerID
// 	_, err = repo.GetByCustomerID(context.Background(), 0)
// 	if !errors.Is(err, repository.ErrInvalidInput) {
// 		log.Fatal("❌ Should validate customerID")
// 	}
// 	fmt.Println("✅ GetByCustomerID validates customerID")

// 	fmt.Println("\n=== Testing GetOrderWithItems ===")

// 	// 1. Создаем тестовый продукт
// 	productRepo := repository.NewProductRepository(conn)
// 	testProduct := &models.Product{
// 		Name:     "Тестовый товар для заказа",
// 		Quantity: 100,
// 		Price:    1500.00,
// 		Category: "Тест",
// 	}
// 	productRepo.Create(context.Background(), testProduct)

// 	// 2. Создаем заказ с товарами напрямую через SQL
// 	var orderID int
// 	conn.QueryRow(context.Background(),
// 		`INSERT INTO orders (customer_id, total_amount, status)
//      VALUES ($1, $2, $3) RETURNING order_id`,
// 		1, 5000.00, "created",
// 	).Scan(&orderID)

// 	// 3. Добавляем товары в заказ
// 	conn.Exec(context.Background(),
// 		`INSERT INTO order_items (order_id, product_id, quantity, price)
//      VALUES ($1, $2, $3, $4)`,
// 		orderID, testProduct.ProductID, 2, 1500.00,
// 	)

// 	conn.Exec(context.Background(),
// 		`INSERT INTO order_items (order_id, product_id, quantity, price)
//      VALUES ($1, $2, $3, $4)`,
// 		orderID, testProduct.ProductID, 1, 2000.00,
// 	)

// 	fmt.Printf("✅ Created order with items: ID %d\n", orderID)

// 	// 4. Тестируем GetOrderWithItems
// 	order, items, err := repo.GetOrderWithItems(context.Background(), orderID)
// 	if err != nil {
// 		log.Fatal("❌ GetOrderWithItems failed:", err)
// 	}

// 	fmt.Printf("✅ Found order: ID %d, Status: %s\n", order.OrderID, order.Status)
// 	fmt.Printf("✅ Found %d items in order:\n", len(items))
// 	for i, item := range items {
// 		fmt.Printf("   %d. ProductID: %d, Quantity: %d, Price: %.2f\n",
// 			i+1, item.ProductID, item.Quantity, item.Price)
// 	}

// 	// 5. Тест: заказ без товаров (создаем новый заказ)
// 	var emptyOrderID int
// 	conn.QueryRow(context.Background(),
// 		`INSERT INTO orders (customer_id, total_amount, status)
//      VALUES ($1, $2, $3) RETURNING order_id`,
// 		1, 1000.00, "created",
// 	).Scan(&emptyOrderID)

// 	_, emptyItems, err := repo.GetOrderWithItems(context.Background(), emptyOrderID)
// 	if err != nil {
// 		log.Fatal("❌ GetOrderWithItems for empty order failed:", err)
// 	}
// 	if len(emptyItems) != 0 {
// 		log.Fatal("❌ Empty order should have 0 items")
// 	}
// 	fmt.Println("✅ Empty order correctly returns 0 items")

// 	// 6. Тест: несуществующий заказ
// 	_, _, err = repo.GetOrderWithItems(context.Background(), 99999)
// 	if err != repository.ErrNotFound {
// 		log.Fatal("❌ Should return ErrNotFound for non-existent order")
// 	}
// 	fmt.Println("✅ Correctly returns ErrNotFound")

// 	fmt.Println("\n🎉 GetOrderWithItems TESTS PASSED!")
// }

// func testCustomerRepository(conn *pgx.Conn) {
// 	repo := repository.NewCustomerRepository(conn)

// 	// //TECT 1: create

// 	customer1 := &models.Customer{
// 		Name:        "Серега",
// 		PhoneNumber: "+79778756912",
// 		Address:     "Russia, Moscow",
// 		Email:       "Vyachesllaavv@gmail.com",
// 	}

// 	err := repo.Create(context.Background(), customer1)
// 	if err != nil {
// 		log.Fatal("Create test failed", err)
// 		return
// 	}
// 	fmt.Printf("Created customer ID: %d\n", customer1.CustomerID)

// 	// // Тест 2: Дубликат email
// 	// customer2 := &models.Customer{
// 	// 	Name:        "Серега",
// 	// 	PhoneNumber: "+79161234567",
// 	// 	Address:     "Tojikistan, Dushanbe",
// 	// 	Email:       "Vyacheslav@gmail.com",
// 	// }

// 	// err = repo.Create(context.Background(), customer2)
// 	// if err == nil || !strings.Contains(err.Error(), "already exist") {
// 	// 	log.Fatal("Should reject duplicate email")
// 	// }
// 	// fmt.Printf("Correctly rejected duplicate email: %v\n", err)

// 	// // Тест 3: Дубликат phone
// 	// customer3 := &models.Customer{
// 	// 	Name:        "Санек",
// 	// 	PhoneNumber: "+79778756980",
// 	// 	Address:     "Uzbekistan, Mbappe",
// 	// 	Email:       "Pidor1488@gmail.com",
// 	// }

// 	// err = repo.Create(context.Background(), customer3)
// 	// if err == nil || !strings.Contains(err.Error(), "already exists") {
// 	// 	log.Fatal("Should reject duplicate phone")
// 	// }
// 	// fmt.Printf("Correctly rejected duplicate phone: %v\n", err)

// 	// //Тест 4: Невалидный email

// 	// customer4 := &models.Customer{
// 	// 	Name:        "Данек",
// 	// 	PhoneNumber: "+79882132233",
// 	// 	Email:       "Xyesos228",
// 	// 	Address:     "Zalupaches",
// 	// }

// 	// err = repo.Create(context.Background(), customer4)
// 	// if err == nil {
// 	// 	log.Fatal("should reject invalid Email", err)
// 	// }
// 	// fmt.Printf("Correctly rejected invalid Email: %v\n", err)
// 	//
// 	//
// 	//
// 	//
// 	// Тест 5: GetByID

// 	savedCustomer, err := repo.GetByID(context.Background(), customer1.CustomerID)
// 	if err != nil {
// 		log.Fatal("GetByID test failed:", err)
// 		return
// 	}
// 	fmt.Printf("Retrieved customer:\n")
// 	fmt.Printf("ID: %d\n", savedCustomer.CustomerID)
// 	fmt.Printf("Name: %s\n", savedCustomer.Name)

// 	// Тест 6: GetByID с несуществующим ID

// 	_, err = repo.GetByID(context.Background(), 99999)
// 	if err != nil {
// 		fmt.Printf("Correctly got error for non-existent product: %v\n", err)
// 	} else {
// 		fmt.Println("Should have gotten error for non-existent product")
// 	}
// 	//
// 	//
// 	//
// 	//

// 	//TECT 5: GetAll
// 	customer5 := &models.Customer{
// 		Name:        "Алексей",
// 		PhoneNumber: "+79161112233",
// 		Email:       "alexey@example.com",
// 	}
// 	repo.Create(context.Background(), customer5)

// 	allCustomers, err := repo.GetAll(context.Background())
// 	if err != nil {
// 		log.Fatal("GetAll failed:", err)
// 	}

// 	fmt.Printf("Found %d customers:\n", len(allCustomers))
// 	for i, c := range allCustomers {
// 		fmt.Printf("  %d. ID: %d, Name: %s, Email: %s\n",
// 			i+1, c.CustomerID, c.Name, c.Email)
// 	}

// 	// Проверяем что получили хотя бы 2 клиента
// 	if len(allCustomers) < 2 {
// 		log.Fatal("Should have at least 2 customers")
// 	}

// 	fmt.Println("\n=== Testing Update ===")

// 	// 1. Получаем клиента
// 	customer, _ := repo.GetByID(context.Background(), customer1.CustomerID)

// 	// 2. Меняем на уникальный email
// 	customer.Email = "updated-unique@example.com"
// 	err = repo.Update(context.Background(), customer)
// 	if err != nil {
// 		log.Fatal("Update with unique email failed:", err)
// 	}
// 	fmt.Println("✅ Update with unique email successful")

// 	// 3. Пробуем изменить другого клиента на тот же email
// 	otherCustomer, _ := repo.GetByID(context.Background(), customer5.CustomerID)
// 	otherCustomer.Email = "updated-unique@example.com" // уже занят!
// 	err = repo.Update(context.Background(), otherCustomer)
// 	if err == nil || !strings.Contains(err.Error(), "already exists") {
// 		log.Fatal("Should reject duplicate email in Update")
// 	}
// 	fmt.Println("✅ Update correctly rejects duplicate email")

// 	// 4. Пробуем обновить несуществующего клиента
// 	fakeCustomer := &models.Customer{
// 		CustomerID:  99999,
// 		Name:        "Fake",
// 		Email:       "fake@example.com",
// 		PhoneNumber: "+79169999999",
// 	}
// 	err = repo.Update(context.Background(), fakeCustomer)
// 	if err != repository.ErrNotFound {
// 		log.Fatal("Should return ErrNotFound for non-existent customer")
// 	}
// 	fmt.Println("✅ Update returns ErrNotFound correctly")

// 	fmt.Println("\n=== Testing Delete ===")

// 	// 1. Создаем клиента для удаления
// 	customerToDelete := &models.Customer{
// 		Name:        "Удаляемый",
// 		PhoneNumber: "+79167778899",
// 		Email:       "to-delete@example.com",
// 	}
// 	repo.Create(context.Background(), customerToDelete)
// 	fmt.Printf("✅ Created customer for deletion ID: %d\n", customerToDelete.CustomerID)

// 	// 2. Удаляем
// 	err = repo.Delete(context.Background(), customerToDelete.CustomerID)
// 	if err != nil {
// 		log.Fatal("Delete failed:", err)
// 	}
// 	fmt.Println("✅ Customer deleted")

// 	// 3. Проверяем что удален
// 	_, err = repo.GetByID(context.Background(), customerToDelete.CustomerID)
// 	if err != repository.ErrNotFound {
// 		log.Fatal("❌ Deleted customer should not be found")
// 	}
// 	fmt.Println("✅ GetByID returns ErrNotFound as expected")

// 	// 4. Пробуем удалить несуществующего
// 	err = repo.Delete(context.Background(), 99999)
// 	if err != repository.ErrNotFound {
// 		log.Fatal("❌ Should return ErrNotFound")
// 	}
// 	fmt.Println("✅ Delete returns ErrNotFound correctly")

// 	// 5. Пробуем удалить с ID=0
// 	err = repo.Delete(context.Background(), 0)
// 	if !errors.Is(err, repository.ErrInvalidInput) {
// 		log.Fatal("❌ Should validate ID")
// 	}
// 	fmt.Println("✅ Delete validates ID correctly")

// 	fmt.Println("\n=== Testing GetByEmail and GetByPhoneNumber ===")

// 	// Тест 1: GetByEmail существующего
// 	customerByEmail, err := repo.GetByEmail(context.Background(), "alexey@example.com")
// 	if err != nil {
// 		log.Fatal("GetByEmail failed:", err)
// 	}
// 	fmt.Printf("✅ Found by email: %s (ID: %d)\n", customerByEmail.Name, customerByEmail.CustomerID)

// 	// Тест 2: GetByEmail несуществующего
// 	_, err = repo.GetByEmail(context.Background(), "nonexistent@example.com")
// 	if err != repository.ErrNotFound {
// 		log.Fatal("Should return ErrNotFound for non-existent email")
// 	}
// 	fmt.Println("✅ GetByEmail returns ErrNotFound correctly")

// 	// Тест 3: GetByEmail с пустой строкой
// 	_, err = repo.GetByEmail(context.Background(), "")
// 	if !errors.Is(err, repository.ErrInvalidInput) {
// 		log.Fatal("Should validate empty email")
// 	}
// 	fmt.Println("✅ GetByEmail validates empty email")

// 	// Тест 4: GetByPhoneNumber существующего
// 	customerByPhone, err := repo.GetByPhoneNumber(context.Background(), "+79161112233")
// 	if err != nil {
// 		log.Fatal("GetByPhoneNumber failed:", err)
// 	}
// 	fmt.Printf("✅ Found by phone: %s (ID: %d)\n", customerByPhone.Name, customerByPhone.CustomerID)

// 	// Тест 5: GetByPhoneNumber несуществующего
// 	_, err = repo.GetByPhoneNumber(context.Background(), "+79999999999")
// 	if err != repository.ErrNotFound {
// 		log.Fatal("Should return ErrNotFound for non-existent phone")
// 	}
// 	fmt.Println("✅ GetByPhoneNumber returns ErrNotFound correctly")

// 	// Тест 6: GetByPhoneNumber с пустой строкой
// 	_, err = repo.GetByPhoneNumber(context.Background(), "")
// 	if !errors.Is(err, repository.ErrInvalidInput) {
// 		log.Fatal("Should validate empty phone")
// 	}
// 	fmt.Println("✅ GetByPhoneNumber validates empty phone")
// }

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
