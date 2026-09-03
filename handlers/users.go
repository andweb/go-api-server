package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"go-api-server/models"
)

// GetUsers returns all users as JSON.
func GetUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stmt, err := db.Prepare(`SELECT id, name, email FROM users`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer stmt.Close()

		rows, err := stmt.Query()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		users := make([]models.User, 0)
		for rows.Next() {
			var u models.User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			users = append(users, u)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, users)
	}
}

// GetUser returns a single user by ID from the URL path.
func GetUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseUserID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var u models.User
		err = db.QueryRow(`SELECT id, name, email FROM users WHERE id = ?`, id).Scan(&u.ID, &u.Name, &u.Email)
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
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
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		stmt, err := db.Prepare(`INSERT INTO users (name, email) VALUES (?, ?)`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(u.Name, u.Email)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		u.ID, err = res.LastInsertId()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusCreated, u)
	}
}

// UpdateUser updates an existing user by ID and returns the updated record.
func UpdateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseUserID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var u models.User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		stmt, err := db.Prepare(`UPDATE users SET name=?, email=? WHERE id=?`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(u.Name, u.Email, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		n, err := res.RowsAffected()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if n == 0 {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}

		u.ID = id
		respondJSON(w, http.StatusOK, u)
	}
}

// DeleteUser deletes a user by ID and responds with 204 No Content.
func DeleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseUserID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		stmt, err := db.Prepare(`DELETE FROM users WHERE id = ?`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		n, err := res.RowsAffected()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if n == 0 {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}
