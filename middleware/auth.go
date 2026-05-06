package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "No token provided", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") || len(authHeader) <= len("Bearer ") {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			jwtSecret = "JWT_SECRET"
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if raw, ok := claims["userId"]; ok {
				switch v := raw.(type) {
				case float64:
					ctx = context.WithValue(ctx, ctxKeyUserID, int(v))
				case int:
					ctx = context.WithValue(ctx, ctxKeyUserID, v)
				case int64:
					ctx = context.WithValue(ctx, ctxKeyUserID, int(v))
				case string:
					ctx = context.WithValue(ctx, ctxKeyUserID, v)
				}
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
