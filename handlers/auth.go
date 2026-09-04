package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"go-api-server/models"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type authResponse struct {
	Token  string `json:"token"`
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
}

type jwtClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func jwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	return []byte(secret)
}

func generateJWT(userID int64, email string) (string, error) {
	claims := jwtClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// Register creates a user from email/password and returns a JWT.
func Register(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}
		if err := models.ValidateEmail(req.Email); err != nil {
			handleError(w, err, http.StatusBadRequest, err.Error())
			return
		}
		if len(req.Password) < 6 {
			handleError(w, nil, http.StatusBadRequest, "password must be at least 6 characters")
			return
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = strings.Split(req.Email, "@")[0]
		}
		if err := models.ValidateUser(&models.User{Name: name, Email: req.Email}); err != nil {
			handleError(w, err, http.StatusBadRequest, err.Error())
			return
		}

		hash, err := models.HashPassword(req.Password)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		res, err := db.Exec(`INSERT INTO users (name, email, password) VALUES (?, ?, ?)`, name, req.Email, hash)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				handleError(w, err, http.StatusConflict, "email already registered")
				return
			}
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		id, err := res.LastInsertId()
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		token, err := generateJWT(id, req.Email)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		respondJSON(w, http.StatusCreated, authResponse{Token: token, UserID: id, Email: req.Email})
	}
}

// Login verifies credentials and returns a JWT.
func Login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req authRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			handleError(w, err, http.StatusBadRequest, "invalid request")
			return
		}
		if req.Email == "" || req.Password == "" {
			handleError(w, nil, http.StatusBadRequest, "invalid request")
			return
		}

		var id int64
		var hash string
		err := db.QueryRow(`SELECT id, password FROM users WHERE email = ?`, req.Email).Scan(&id, &hash)
		if err == sql.ErrNoRows {
			handleError(w, err, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}
		if !models.CheckPassword(hash, req.Password) {
			handleError(w, nil, http.StatusUnauthorized, "invalid credentials")
			return
		}

		token, err := generateJWT(id, req.Email)
		if err != nil {
			handleError(w, err, http.StatusInternalServerError, "")
			return
		}

		respondJSON(w, http.StatusOK, authResponse{Token: token, UserID: id, Email: req.Email})
	}
}
