package main

import (
	"log"
	"net/http"

	"go-api-server/handlers"
	"go-api-server/middleware"
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

	mux.HandleFunc("POST /register", handlers.Register(db))
	mux.HandleFunc("POST /login", handlers.Login(db))

	mux.HandleFunc("GET /users", handlers.GetUsers(db))
	mux.HandleFunc("GET /users/{id}", handlers.GetUser(db))
	mux.Handle("POST /users", middleware.AuthMiddleware(handlers.CreateUser(db)))
	mux.Handle("PUT /users/{id}", middleware.AuthMiddleware(handlers.UpdateUser(db)))
	mux.Handle("DELETE /users/{id}", middleware.AuthMiddleware(handlers.DeleteUser(db)))

	mux.HandleFunc("GET /users/{id}/orders", handlers.GetUserOrders(db))
	mux.Handle("POST /orders", middleware.AuthMiddleware(handlers.CreateOrder(db)))
	mux.Handle("DELETE /orders/{id}", middleware.AuthMiddleware(handlers.DeleteOrder(db)))

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
