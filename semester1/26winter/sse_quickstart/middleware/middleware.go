// Package middleware implements CORS and JWT auth
package middleware

import (
	"context"
	"errors"
	"net/http"

	"sse/utils"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "userID"

func CORS(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Last-Event-ID")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.URL.Query().Get("token")
		if tokenStr == "" {
			authHeader := r.Header.Get("Authorization")
			tokenStr = utils.ParseAuthHeader(authHeader)
		}

		if tokenStr == "" {
			http.Error(w, "Unauthorized: No token found", http.StatusUnauthorized)
			return
		}

		userID, err := utils.ParseToken(tokenStr)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				http.Error(w, "Expired token", http.StatusUnauthorized)
				return
			} else {
				http.Error(w, "Error token", http.StatusUnauthorized)
				return
			}
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
