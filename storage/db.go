package storage

import (
	"database/sql"
	"os"
	"path/filepath"

	"go-api-server/models"
	_ "modernc.org/sqlite"
)

var (
	_ models.User
	_ models.Order
)

func InitDB() (*sql.DB, error) {
	if err := os.MkdirAll("data", 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", filepath.Join("data", "shop.db"))
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	if err := Migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

const Schema = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	email TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS products (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	price REAL NOT NULL
);
CREATE TABLE IF NOT EXISTS orders (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	product TEXT NOT NULL,
	quantity INTEGER NOT NULL,
	price REAL NOT NULL,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`

func Migrate(db *sql.DB) error {
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return err
	}
	_, err := db.Exec(Schema)
	return err
}

func CloseDB(db *sql.DB) {
	if db != nil {
		_ = db.Close()
	}
}
