package middleware

import (
	"context"
	"net/http"
	"strings"

	"nesta/internal/auth"
	"nesta/internal/http/response"
)

type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyRole   contextKey = "role"
)

func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "missing token", RequestID: GetRequestID(r.Context())})
				return
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "invalid authorization header", RequestID: GetRequestID(r.Context())})
				return
			}

			claims, err := auth.ParseToken(secret, parts[1])
			if err != nil {
				response.ErrorJSON(w, http.StatusUnauthorized, response.Error{Code: "UNAUTHORIZED", Message: "invalid token", RequestID: GetRequestID(r.Context())})
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, claims.Subject)
			ctx = context.WithValue(ctx, contextKeyRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			current, _ := RoleFromContext(r.Context())
			if current != role {
				response.ErrorJSON(w, http.StatusForbidden, response.Error{Code: "FORBIDDEN", Message: "insufficient permissions", RequestID: GetRequestID(r.Context())})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	value := ctx.Value(contextKeyUserID)
	id, ok := value.(string)
	return id, ok && id != ""
}

func RoleFromContext(ctx context.Context) (string, bool) {
	value := ctx.Value(contextKeyRole)
	role, ok := value.(string)
	return role, ok && role != ""
}
