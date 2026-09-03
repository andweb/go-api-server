package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"go-api-server/models"
)

type ordersPage struct {
	Data   []models.Order `json:"data"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// GetUserOrders returns a paginated list of orders for the user ID from the URL path.
func GetUserOrders(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := parseUserID(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}

		limit := 20
		offset := 0

		q := r.URL.Query()
		if raw := q.Get("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				handleError(w, err, http.StatusBadRequest, "invalid request")
				return
			}
			limit = n
		}
		if raw := q.Get("offset"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				handleError(w, err, http.StatusBadRequest, "invalid request")
				return
			}
			offset = n
		}
		if limit > 100 {
			handleError(w, nil, http.StatusBadRequest, "limit cannot exceed 100")
			return
		}

		var exists int64
		err = db.QueryRow(`SELECT id FROM users WHERE id = ?`, userID).Scan(&exists)
		if err == sql.ErrNoRows {
			handleError(w, err, http.StatusNotFound, "")
			return
		}
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		var total int
		if err := db.QueryRow(`SELECT COUNT(*) FROM orders WHERE user_id = ?`, userID).Scan(&total); err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		stmt, err := db.Prepare(`SELECT id, user_id, product, quantity, price FROM orders WHERE user_id = ? LIMIT ? OFFSET ?`)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		defer stmt.Close()

		rows, err := stmt.Query(userID, limit, offset)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		defer rows.Close()

		orders := make([]models.Order, 0)
		for rows.Next() {
			var o models.Order
			if err := rows.Scan(&o.ID, &o.UserID, &o.Product, &o.Quantity, &o.Price); err != nil {
				handleError(w, err, http.StatusInternalServerError, "")
				return
			}
			orders = append(orders, o)
		}
		if err := rows.Err(); err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		respondJSON(w, http.StatusOK, ordersPage{
			Data:   orders,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		})
	}
}

// CreateOrder creates an order from the JSON body after checking that the user exists.
func CreateOrder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var o models.Order
		if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}

		var exists int64
		err := db.QueryRow(`SELECT id FROM users WHERE id = ?`, o.UserID).Scan(&exists)
		if err == sql.ErrNoRows {
			handleError(w, err, http.StatusNotFound, "")
			return
		}
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		stmt, err := db.Prepare(`INSERT INTO orders (user_id, product, quantity, price) VALUES (?, ?, ?, ?)`)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(o.UserID, o.Product, o.Quantity, o.Price)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		o.ID, err = res.LastInsertId()
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		respondJSON(w, http.StatusCreated, o)
	}
}

// DeleteOrder deletes an order by ID and responds with 204, or 404 if missing.
func DeleteOrder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseUserID(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}

		stmt, err := db.Prepare(`DELETE FROM orders WHERE id = ?`)
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
