package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"go-api-server/models"
)

type usersPage struct {
	Data   []models.User `json:"data"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// GetUsers returns a paginated list of users as JSON.
func GetUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, offset, err := parsePagination(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest, err.Error())
			return
		}

		var total int
		if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		stmt, err := db.Prepare(`SELECT id, name, email FROM users LIMIT ? OFFSET ?`)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		defer stmt.Close()

		rows, err := stmt.Query(limit, offset)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		defer rows.Close()

		users := make([]models.User, 0)
		for rows.Next() {
			var u models.User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
				handleError(w, err, http.StatusInternalServerError, "")
				return
			}
			users = append(users, u)
		}
		if err := rows.Err(); err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		respondJSON(w, http.StatusOK, usersPage{
			Data:   users,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		})
	}
}

// GetUser returns a single user by ID from the URL path.
func GetUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseUserID(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}

		var u models.User
		err = db.QueryRow(`SELECT id, name, email FROM users WHERE id = ?`, id).Scan(&u.ID, &u.Name, &u.Email)
		if err == sql.ErrNoRows {
			handleError(w, err, http.StatusNotFound, "")
			return
		}
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		respondJSON(w, http.StatusOK, u)
	}
}

// CreateUser creates a user from the JSON request body and returns it with status 201.
func CreateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u models.User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}
		if err := models.ValidateUser(&u); err != nil {
			handleError(w, err, http.StatusBadRequest, err.Error())
			return
		}
		if len(u.Password) < 6 {
			handleError(w, nil, http.StatusBadRequest, "password must be at least 6 characters")
			return
		}

		hash, err := models.HashPassword(u.Password)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		stmt, err := db.Prepare(`INSERT INTO users (name, email, password) VALUES (?, ?, ?)`)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(u.Name, u.Email, hash)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		u.ID, err = res.LastInsertId()
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		u.Password = ""
		respondJSON(w, http.StatusCreated, u)
	}
}

// UpdateUser updates an existing user by ID and returns the updated record.
func UpdateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseUserID(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}

		var u models.User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}
		if err := models.ValidateUser(&u); err != nil {
			handleError(w, err, http.StatusBadRequest, err.Error())
			return
		}

		stmt, err := db.Prepare(`UPDATE users SET name=?, email=? WHERE id=?`)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(u.Name, u.Email, id)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		n, err := res.RowsAffected()
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		if n == 0 {
			handleError(w, sql.ErrNoRows, http.StatusNotFound, "")
			return
		}

		u.ID = id
		u.Password = ""
		respondJSON(w, http.StatusOK, u)
	}
}

// DeleteUser deletes a user by ID and responds with 204 No Content.
func DeleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseUserID(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}

		stmt, err := db.Prepare(`DELETE FROM users WHERE id = ?`)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(id)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		n, err := res.RowsAffected()
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		if n == 0 {
			handleError(w, sql.ErrNoRows, http.StatusNotFound, "")
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}
