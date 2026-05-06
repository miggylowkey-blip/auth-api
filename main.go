package main

import (
	"auth-api/db"
	"auth-api/handlers"
	"auth-api/middleware"
	"fmt"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load()

	db.Connect()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", handlers.Register)
	mux.HandleFunc("POST /login", handlers.Login)
	mux.Handle("GET /profile", middleware.AuthMiddleware(http.HandlerFunc(handlers.Profile)))

	fmt.Println("Server Running on 8080")
	var h http.Handler = mux
	h = middleware.WithRequestID(h)
	h = middleware.RateLimitMiddleware(h)
	h = middleware.AuditMiddleware(h)
	http.ListenAndServe(":8080", h)
}
