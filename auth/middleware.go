package auth

import (
	"context"
	"net/http"
)

type contextKey string

const UserContextKey = contextKey("user")

func AuthMiddleware(f http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, tok := r.URL.Query()["bearer"]
		if tok && len(token) == 1 {

			user, err := ValidateToken(token[0])
			if err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)

			} else {
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				f(w, r.WithContext(ctx))
			}

		}
	})
}
