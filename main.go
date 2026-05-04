package main

import (
	"auth-api/db"
	"auth-api/handlers"
	"auth-api/middleware"
	"fmt"
	"net/http"
)

func main() {
	db.Connect()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /register", handlers.Register)
	mux.HandleFunc("POST /login", handlers.Login)
	mux.Handle("GET /profile", middleware.AuthMiddleware(http.HandlerFunc(handlers.Profile)))

	fmt.Println("Server Running on 8080")
	http.ListenAndServe(":8080", mux)
}
