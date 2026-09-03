package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-api-server/models"
	"go-api-server/storage"
)

func insertOrder(t *testing.T, db *sql.DB, userID int64, product string, quantity int, price float64) models.Order {
	t.Helper()
	if storage.Schema == "" {
		t.Fatal("empty schema")
	}

	res, err := db.Exec(
		`INSERT INTO orders (user_id, product, quantity, price) VALUES (?, ?, ?, ?)`,
		userID, product, quantity, price,
	)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return models.Order{ID: id, UserID: userID, Product: product, Quantity: quantity, Price: price}
}

func TestGetUserOrders(t *testing.T) {
	t.Run("with orders", func(t *testing.T) {
		db := setupTestDB(t)
		user := insertUser(t, db, "Alice", "alice@example.com")
		other := insertUser(t, db, "Bob", "bob@example.com")
		want := []models.Order{
			insertOrder(t, db, user.ID, "Laptop", 1, 1000),
			insertOrder(t, db, user.ID, "Mouse", 2, 50),
		}
		insertOrder(t, db, other.ID, "Phone", 1, 500)

		mux := http.NewServeMux()
		mux.HandleFunc("GET /users/{id}/orders", GetUserOrders(db))

		req := httptest.NewRequest(http.MethodGet, "/users/1/orders", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var got []models.Order
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Fatalf("len(orders) = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("orders[%d] = %+v, want %+v", i, got[i], want[i])
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		db := setupTestDB(t)
		insertUser(t, db, "Alice", "alice@example.com")

		mux := http.NewServeMux()
		mux.HandleFunc("GET /users/{id}/orders", GetUserOrders(db))

		req := httptest.NewRequest(http.MethodGet, "/users/1/orders", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var got []models.Order
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("orders = %#v, want empty slice", got)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("GET /users/{id}/orders", GetUserOrders(db))

		req := httptest.NewRequest(http.MethodGet, "/users/99/orders", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("GET /users/{id}/orders", GetUserOrders(db))

		req := httptest.NewRequest(http.MethodGet, "/users/abc/orders", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})
}

func TestCreateOrder(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		db := setupTestDB(t)
		user := insertUser(t, db, "Alice", "alice@example.com")

		mux := http.NewServeMux()
		mux.HandleFunc("POST /orders", CreateOrder(db))

		payload := models.Order{UserID: user.ID, Product: "Laptop", Quantity: 2, Price: 1000}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
		}

		var got models.Order
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.ID == 0 {
			t.Error("id = 0, want non-zero")
		}
		if got.UserID != payload.UserID || got.Product != payload.Product || got.Quantity != payload.Quantity || got.Price != payload.Price {
			t.Errorf("order = %+v, want user_id=%d product=%q quantity=%d price=%v", got, payload.UserID, payload.Product, payload.Quantity, payload.Price)
		}

		var stored models.Order
		err = db.QueryRow(`SELECT id, user_id, product, quantity, price FROM orders WHERE id = ?`, got.ID).
			Scan(&stored.ID, &stored.UserID, &stored.Product, &stored.Quantity, &stored.Price)
		if err != nil {
			t.Fatalf("order not stored: %v", err)
		}
		if stored != got {
			t.Errorf("stored = %+v, want %+v", stored, got)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("POST /orders", CreateOrder(db))

		body, err := json.Marshal(models.Order{UserID: 99, Product: "Laptop", Quantity: 1, Price: 1000})
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
		}

		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("orders count = %d, want 0", count)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		db := setupTestDB(t)
		insertUser(t, db, "Alice", "alice@example.com")

		mux := http.NewServeMux()
		mux.HandleFunc("POST /orders", CreateOrder(db))

		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})
}

func TestDeleteOrder(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db := setupTestDB(t)
		user := insertUser(t, db, "Alice", "alice@example.com")
		created := insertOrder(t, db, user.ID, "Laptop", 1, 1000)

		mux := http.NewServeMux()
		mux.HandleFunc("DELETE /orders/{id}", DeleteOrder(db))

		req := httptest.NewRequest(http.MethodDelete, "/orders/1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
		}

		var id int64
		err := db.QueryRow(`SELECT id FROM orders WHERE id = ?`, created.ID).Scan(&id)
		if err != sql.ErrNoRows {
			t.Errorf("expected order deleted, got id=%d err=%v", id, err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("DELETE /orders/{id}", DeleteOrder(db))

		req := httptest.NewRequest(http.MethodDelete, "/orders/99", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("DELETE /orders/{id}", DeleteOrder(db))

		req := httptest.NewRequest(http.MethodDelete, "/orders/abc", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})
}
