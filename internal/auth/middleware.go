package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/mostdoc/mostdoc/internal/httputil"
)

// Middleware extracts and validates the JWT from the Authorization header.
func Middleware(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			user, err := service.ParseAccessToken(token)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth wraps a handler and returns 401 if no valid user is in context.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); !ok {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		next(w, r)
	}
}

// UserFromContext extracts the UserInfo from the request context.
func UserFromContext(ctx context.Context) (UserInfo, bool) {
	user, ok := ctx.Value(UserContextKey).(UserInfo)
	return user, ok
}

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}
