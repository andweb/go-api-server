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

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err := storage.Migrate(db); err != nil {
		db.Close()
		t.Fatal(err)
	}

	t.Cleanup(func() {
		storage.CloseDB(db)
	})
	return db
}

func insertUser(t *testing.T, db *sql.DB, name, email string) models.User {
	t.Helper()

	res, err := db.Exec(`INSERT INTO users (name, email) VALUES (?, ?)`, name, email)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return models.User{ID: id, Name: name, Email: email}
}

func TestGetUsers(t *testing.T) {
	db := setupTestDB(t)
	want := []models.User{
		insertUser(t, db, "Alice", "alice@example.com"),
		insertUser(t, db, "Bob", "bob@example.com"),
		insertUser(t, db, "Carol", "carol@example.com"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users", GetUsers(db))

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got []models.User
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len(users) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("users[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestGetUser(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db := setupTestDB(t)
		want := insertUser(t, db, "Alice", "alice@example.com")

		mux := http.NewServeMux()
		mux.HandleFunc("GET /users/{id}", GetUser(db))

		req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var got models.User
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("user = %+v, want %+v", got, want)
		}
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("GET /users/{id}", GetUser(db))

		req := httptest.NewRequest(http.MethodGet, "/users/99", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("GET /users/{id}", GetUser(db))

		req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})
}

func TestCreateUser(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("POST /users", CreateUser(db))

		payload := models.User{Name: "Alice", Email: "alice@example.com"}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
		}

		var got models.User
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.ID == 0 {
			t.Error("id = 0, want non-zero")
		}
		if got.Name != payload.Name || got.Email != payload.Email {
			t.Errorf("user = %+v, want name=%q email=%q", got, payload.Name, payload.Email)
		}

		var stored models.User
		err = db.QueryRow(`SELECT id, name, email FROM users WHERE id = ?`, got.ID).Scan(&stored.ID, &stored.Name, &stored.Email)
		if err != nil {
			t.Fatalf("user not stored: %v", err)
		}
		if stored != got {
			t.Errorf("stored = %+v, want %+v", stored, got)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("POST /users", CreateUser(db))

		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db := setupTestDB(t)
		created := insertUser(t, db, "Alice", "alice@example.com")

		mux := http.NewServeMux()
		mux.HandleFunc("PUT /users/{id}", UpdateUser(db))

		payload := models.User{Name: "Alice Updated", Email: "alice.updated@example.com"}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var got models.User
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.ID != created.ID {
			t.Errorf("id = %d, want %d", got.ID, created.ID)
		}
		if got.Name != payload.Name || got.Email != payload.Email {
			t.Errorf("user = %+v, want name=%q email=%q", got, payload.Name, payload.Email)
		}

		var stored models.User
		err = db.QueryRow(`SELECT id, name, email FROM users WHERE id = ?`, created.ID).Scan(&stored.ID, &stored.Name, &stored.Email)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Name != payload.Name || stored.Email != payload.Email {
			t.Errorf("stored = %+v, want name=%q email=%q", stored, payload.Name, payload.Email)
		}
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("PUT /users/{id}", UpdateUser(db))

		body, err := json.Marshal(models.User{Name: "Ghost", Email: "ghost@example.com"})
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPut, "/users/99", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("PUT /users/{id}", UpdateUser(db))

		req := httptest.NewRequest(http.MethodPut, "/users/abc", bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		db := setupTestDB(t)
		insertUser(t, db, "Alice", "alice@example.com")

		mux := http.NewServeMux()
		mux.HandleFunc("PUT /users/{id}", UpdateUser(db))

		req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewBufferString("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		db := setupTestDB(t)
		created := insertUser(t, db, "Alice", "alice@example.com")

		mux := http.NewServeMux()
		mux.HandleFunc("DELETE /users/{id}", DeleteUser(db))

		req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
		}

		var id int64
		err := db.QueryRow(`SELECT id FROM users WHERE id = ?`, created.ID).Scan(&id)
		if err != sql.ErrNoRows {
			t.Errorf("expected user deleted, got id=%d err=%v", id, err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("DELETE /users/{id}", DeleteUser(db))

		req := httptest.NewRequest(http.MethodDelete, "/users/99", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("DELETE /users/{id}", DeleteUser(db))

		req := httptest.NewRequest(http.MethodDelete, "/users/abc", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})
}
