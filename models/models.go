package models

import (
	"errors"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
}

type Order struct {
	ID       int64   `json:"id"`
	UserID   int64   `json:"user_id"`
	Product  string  `json:"product"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

func ValidateUser(u *User) error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	if len(u.Name) < 2 || len(u.Name) > 100 {
		return errors.New("name is required")
	}
	return ValidateEmail(u.Email)
}

func ValidateEmail(email string) error {
	if !emailRegexp.MatchString(email) {
		return errors.New("email is invalid")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
