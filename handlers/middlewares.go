package handlers

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	ContextUsernameKey = contextKey("username")
	ContextRoleKey     = contextKey("role")
)

func OnlyAdminMiddleware(next ApiHandlerFunc) ApiHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
		// Get the role from the context
		role := r.Context().Value(ContextRoleKey).(string)
		if role != "admin" {
			return nil, &HandlerError{Status: http.StatusForbidden, Message: ErrorResponse{Code: "E403", Message: "Forbidden", Detail: "You are not an admin"}}
		}
		return next(w, r)
	}
}

func JWTAuthMiddleware(next ApiHandlerFunc) ApiHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) (*HandlerSuccess, *HandlerError) {
		authHeader := r.Header.Get("Authorization")

		// Check if the Authorization header is present
		if authHeader == "" {
			return nil, &HandlerError{Status: http.StatusUnauthorized, Message: ErrorResponse{Code: "E401", Message: "Unauthorized", Detail: "Missing token"}}
		}

		// Token should be in the format: "Bearer <Token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return nil, &HandlerError{Status: http.StatusUnauthorized, Message: ErrorResponse{Code: "E401", Message: "Unauthorized", Detail: "Invalid token format"}}
		}

		// Verify the token
		tokenSting := parts[1]
		claims, err := VerifyJwtToken(tokenSting)
		if err != nil {
			return nil, &HandlerError{Status: http.StatusUnauthorized, Message: ErrorResponse{Code: "E401", Message: "Unauthorized", Detail: "Invalid token"}}
		}

		// Get the username and role from the claims and store them in the request context
		ctx := context.WithValue(r.Context(), ContextUsernameKey, claims["username"].(string))
		ctx = context.WithValue(ctx, ContextRoleKey, claims["role"].(string))

		r = r.WithContext(ctx)
		next(w, r)

		return &HandlerSuccess{Status: http.StatusOK, Data: nil}, nil
	}

}
