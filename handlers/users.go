package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"go-api-server/models"
)

type UsersPage struct {
	Data   []models.User `json:"data"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

// GetUsers godoc
// @Summary      List users
// @Description  Returns a paginated list of users
// @Tags         Users
// @Produce      json
// @Param        limit   query     int  false  "Page size (1-100)"  default(20)
// @Param        offset  query     int  false  "Offset (>=0)"       default(0)
// @Success      200     {object}  UsersPage
// @Failure      400     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /users [get]
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

		respondJSON(w, http.StatusOK, UsersPage{
			Data:   users,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		})
	}
}

// GetUser godoc
// @Summary      Get user
// @Description  Returns a single user by ID
// @Tags         Users
// @Produce      json
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  models.User
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /users/{id} [get]
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

// CreateUser godoc
// @Summary      Create user
// @Description  Creates a user (requires JWT)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        user  body      models.User  true  "User payload (name, email, password)"
// @Success      201   {object}  models.User
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users [post]
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

// UpdateUser godoc
// @Summary      Update user
// @Description  Updates a user by ID (requires JWT)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id    path      int          true  "User ID"
// @Param        user  body      models.User  true  "User payload (name, email)"
// @Success      200   {object}  models.User
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [put]
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

// DeleteUser godoc
// @Summary      Delete user
// @Description  Deletes a user by ID (requires JWT)
// @Tags         Users
// @Produce      json
// @Param        id   path  int  true  "User ID"
// @Success      204  "No Content"
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /users/{id} [delete]
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
