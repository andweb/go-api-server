package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"go-api-server/models"
)

// GetUserOrders returns all orders for the user ID from the URL path.
func GetUserOrders(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := parseUserID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		var exists int64
		err = db.QueryRow(`SELECT id FROM users WHERE id = ?`, userID).Scan(&exists)
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		stmt, err := db.Prepare(`SELECT id, user_id, product, quantity, price FROM orders WHERE user_id = ?`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer stmt.Close()

		rows, err := stmt.Query(userID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer rows.Close()

		orders := make([]models.Order, 0)
		for rows.Next() {
			var o models.Order
			if err := rows.Scan(&o.ID, &o.UserID, &o.Product, &o.Quantity, &o.Price); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			orders = append(orders, o)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondJSON(w, http.StatusOK, orders)
	}
}

// CreateOrder creates an order from the JSON body after checking that the user exists.
func CreateOrder(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var o models.Order
		if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		var exists int64
		err := db.QueryRow(`SELECT id FROM users WHERE id = ?`, o.UserID).Scan(&exists)
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		stmt, err := db.Prepare(`INSERT INTO orders (user_id, product, quantity, price) VALUES (?, ?, ?, ?)`)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer stmt.Close()

		res, err := stmt.Exec(o.UserID, o.Product, o.Quantity, o.Price)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		o.ID, err = res.LastInsertId()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
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
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		stmt, err := db.Prepare(`DELETE FROM orders WHERE id = ?`)
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
			writeError(w, http.StatusNotFound, "order not found")
			return
		}

		respondJSON(w, http.StatusNoContent, nil)
	}
}
