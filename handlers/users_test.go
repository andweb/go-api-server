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

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, status int, msg string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, status, rec.Body.String())
	}
	var body ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error != msg {
		t.Errorf("error = %q, want %q", body.Error, msg)
	}
	if body.Code != status {
		t.Errorf("code = %d, want %d", body.Code, status)
	}
	if body.Timestamp == "" {
		t.Error("timestamp is empty")
	}
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

	var got usersPage
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Total != len(want) {
		t.Errorf("total = %d, want %d", got.Total, len(want))
	}
	if got.Limit != 20 || got.Offset != 0 {
		t.Errorf("limit/offset = %d/%d, want 20/0", got.Limit, got.Offset)
	}
	if len(got.Data) != len(want) {
		t.Fatalf("len(data) = %d, want %d", len(got.Data), len(want))
	}
	for i := range want {
		if got.Data[i] != want[i] {
			t.Errorf("data[%d] = %+v, want %+v", i, got.Data[i], want[i])
		}
	}
}

func TestGetUsersPagination(t *testing.T) {
	db := setupTestDB(t)
	users := []models.User{
		insertUser(t, db, "User1", "u1@example.com"),
		insertUser(t, db, "User2", "u2@example.com"),
		insertUser(t, db, "User3", "u3@example.com"),
		insertUser(t, db, "User4", "u4@example.com"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users", GetUsers(db))

	t.Run("first page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users?limit=2&offset=0", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var got usersPage
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.Total != 4 || got.Limit != 2 || got.Offset != 0 {
			t.Errorf("meta = total=%d limit=%d offset=%d, want 4/2/0", got.Total, got.Limit, got.Offset)
		}
		if len(got.Data) != 2 {
			t.Fatalf("len(data) = %d, want 2", len(got.Data))
		}
		if got.Data[0] != users[0] || got.Data[1] != users[1] {
			t.Errorf("data = %+v, want first two %+v", got.Data, users[:2])
		}
	})

	t.Run("second page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users?limit=2&offset=2", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var got usersPage
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.Total != 4 || got.Limit != 2 || got.Offset != 2 {
			t.Errorf("meta = total=%d limit=%d offset=%d, want 4/2/2", got.Total, got.Limit, got.Offset)
		}
		if len(got.Data) != 2 {
			t.Fatalf("len(data) = %d, want 2", len(got.Data))
		}
		if got.Data[0] != users[2] || got.Data[1] != users[3] {
			t.Errorf("data = %+v, want next two %+v", got.Data, users[2:])
		}
	})

	t.Run("limit too large", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users?limit=200", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusBadRequest, "limit cannot exceed 100")
	})

	t.Run("invalid limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users?limit=abc", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusBadRequest, "invalid request")
	})
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

		assertErrorResponse(t, rec, http.StatusNotFound, "not found")
	})

	t.Run("invalid id", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("GET /users/{id}", GetUser(db))

		req := httptest.NewRequest(http.MethodGet, "/users/abc", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusBadRequest, "invalid request")
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

		assertErrorResponse(t, rec, http.StatusBadRequest, "invalid request")
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

		assertErrorResponse(t, rec, http.StatusNotFound, "not found")
	})

	t.Run("invalid id", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("PUT /users/{id}", UpdateUser(db))

		req := httptest.NewRequest(http.MethodPut, "/users/abc", bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusBadRequest, "invalid request")
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

		assertErrorResponse(t, rec, http.StatusBadRequest, "invalid request")
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

		assertErrorResponse(t, rec, http.StatusNotFound, "not found")
	})

	t.Run("invalid id", func(t *testing.T) {
		db := setupTestDB(t)

		mux := http.NewServeMux()
		mux.HandleFunc("DELETE /users/{id}", DeleteUser(db))

		req := httptest.NewRequest(http.MethodDelete, "/users/abc", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assertErrorResponse(t, rec, http.StatusBadRequest, "invalid request")
	})
}
