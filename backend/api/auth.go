package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// AuthMiddleware returns middleware that validates Bearer token authentication.
func AuthMiddleware(password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if password == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")

			if subtle.ConstantTimeCompare([]byte(token), []byte(password)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
