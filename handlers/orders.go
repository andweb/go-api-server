package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"go-api-server/models"
)

type OrdersPage struct {
	Data   []models.Order `json:"data"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// GetUserOrders godoc
// @Summary      List user orders
// @Description  Returns a paginated list of orders for a user
// @Tags         Orders
// @Produce      json
// @Param        id      path      int  true   "User ID"
// @Param        limit   query     int  false  "Page size (1-100)"  default(20)
// @Param        offset  query     int  false  "Offset (>=0)"       default(0)
// @Success      200     {object}  OrdersPage
// @Failure      400     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /users/{id}/orders [get]
func GetUserOrders(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := parseUserID(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}

		limit, offset, err := parsePagination(r)
		if err != nil {
			handleError(w, err, http.StatusBadRequest, err.Error())
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

		respondJSON(w, http.StatusOK, OrdersPage{
			Data:   orders,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		})
	}
}

// CreateOrder godoc
// @Summary      Create order
// @Description  Creates an order (requires JWT)
// @Tags         Orders
// @Accept       json
// @Produce      json
// @Param        order  body      models.Order  true  "Order payload"
// @Success      201    {object}  models.Order
// @Failure      400    {object}  ErrorResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /orders [post]
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

// DeleteOrder godoc
// @Summary      Delete order
// @Description  Deletes an order by ID (requires JWT)
// @Tags         Orders
// @Produce      json
// @Param        id   path  int  true  "Order ID"
// @Success      204  "No Content"
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /orders/{id} [delete]
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
