package main

import (
	"log"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"go-api-server/handlers"
	"go-api-server/middleware"
	"go-api-server/storage"

	_ "go-api-server/docs"
)

// @title           go-api-server API
// @version         1.0
// @description     REST API on Go with SQLite, JWT auth and pagination.
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
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

	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	addr := ":8080"
	log.Printf("listening on %s", addr)
	log.Printf("swagger UI: http://localhost%s/swagger/index.html", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
