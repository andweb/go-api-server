package main

import (
	"log"
	"net/http"

	"go-api-server/handlers"
	"go-api-server/storage"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func main() {
	db, err := storage.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer storage.CloseDB(db)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /users", handlers.GetUsers(db))
	mux.HandleFunc("GET /users/{id}", handlers.GetUser(db))
	mux.HandleFunc("POST /users", handlers.CreateUser(db))
	mux.HandleFunc("PUT /users/{id}", handlers.UpdateUser(db))
	mux.HandleFunc("DELETE /users/{id}", handlers.DeleteUser(db))
	mux.HandleFunc("GET /users/{id}/orders", handlers.GetUserOrders(db))
	mux.HandleFunc("POST /orders", handlers.CreateOrder(db))
	mux.HandleFunc("DELETE /orders/{id}", handlers.DeleteOrder(db))

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
