package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "user_id"

type claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type errorBody struct {
	Error     string `json:"error"`
	Code      int    `json:"code"`
	Timestamp string `json:"timestamp"`
}

func jwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	return []byte(secret)
}

// AuthMiddleware checks Authorization: Bearer <token> and stores user_id in context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if raw == "" {
			writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}

		parsed, err := jwt.ParseWithClaims(raw, &claims{}, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrTokenSignatureInvalid
			}
			return jwtSecret(), nil
		})
		if err != nil || !parsed.Valid {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		c, ok := parsed.Claims.(*claims)
		if !ok || c.UserID == 0 {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, c.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserIDFromContext returns the authenticated user id from the request context.
func GetUserIDFromContext(r *http.Request) (int64, bool) {
	id, ok := r.Context().Value(userIDKey).(int64)
	return id, ok
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{
		Error:     msg,
		Code:      status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
