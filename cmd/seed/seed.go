package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand/v2"

	"go-api-server/storage"
)

func main() {
	db, err := storage.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer storage.CloseDB(db)

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		log.Fatal(err)
	}
	if count > 0 {
		fmt.Println("Database already seeded")
		return
	}

	if err := seed(db); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Seed completed: 20 users, 10 products, 50 orders")
}

func seed(db *sql.DB) error {
	products := []struct {
		name  string
		price float64
	}{
		{"Laptop", 1000},
		{"Phone", 500},
		{"Tablet", 700},
		{"Headphones", 150},
		{"Monitor", 300},
		{"Keyboard", 100},
		{"Mouse", 50},
		{"Charger", 80},
		{"Speaker", 200},
		{"Camera", 400},
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	userStmt, err := tx.Prepare(`INSERT INTO users (name, email) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer userStmt.Close()

	for i := 1; i <= 20; i++ {
		if _, err := userStmt.Exec(fmt.Sprintf("User %d", i), fmt.Sprintf("user%d@examle.com", i)); err != nil {
			return err
		}
	}

	productStmt, err := tx.Prepare(`INSERT INTO products (name, price) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer productStmt.Close()

	for _, p := range products {
		if _, err := productStmt.Exec(p.name, p.price); err != nil {
			return err
		}
	}

	orderStmt, err := tx.Prepare(`INSERT INTO orders (user_id, product, quantity, price) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer orderStmt.Close()

	for i := 0; i < 50; i++ {
		userID := rand.IntN(20) + 1
		product := products[rand.IntN(len(products))].name
		quantity := rand.IntN(5) + 1
		price := float64(rand.IntN(951) + 50)
		if _, err := orderStmt.Exec(userID, product, quantity, price); err != nil {
			return err
		}
	}

	return tx.Commit()
}
