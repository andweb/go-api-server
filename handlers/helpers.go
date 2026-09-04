package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"
)

// ErrorResponse is the unified JSON error body for all handlers.
type ErrorResponse struct {
	Error     string `json:"error"`
	Code      int    `json:"code"`
	Timestamp string `json:"timestamp"`
}

func handleError(w http.ResponseWriter, err error, status int, msg string) {
	log.Printf("ERROR: %v", err)

	if msg == "" {
		msg = statusMessage(status)
	}

	respondJSON(w, status, ErrorResponse{
		Error:     msg,
		Code:      status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func statusMessage(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad request"
	case http.StatusNotFound:
		return "not found"
	case http.StatusInternalServerError:
		return "internal error"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusConflict:
		return "conflict"
	default:
		return http.StatusText(status)
	}
}

func parseUserID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func parsePagination(r *http.Request) (limit, offset int, err error) {
	limit = 20
	offset = 0

	q := r.URL.Query()
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("invalid request")
		}
		limit = n
	}
	if raw := q.Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("invalid request")
		}
		offset = n
	}
	if limit < 1 {
		return 0, 0, errors.New("limit must be at least 1")
	}
	if limit > 100 {
		return 0, 0, errors.New("limit cannot exceed 100")
	}
	if offset < 0 {
		return 0, 0, errors.New("offset cannot be negative")
	}
	return limit, offset, nil
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(data)
}
