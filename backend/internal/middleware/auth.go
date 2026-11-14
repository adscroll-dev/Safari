// Package middleware provides HTTP middleware: JWT authentication
// and JSON / CORS helpers used by all handlers.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/vyro/backend/internal/auth"
)

type ctxKey string

const userIDKey ctxKey = "userID"

// UserID returns the authenticated user id from the request context.
func UserID(r *http.Request) string {
	if v, ok := r.Context().Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// WriteJSON writes a JSON response with the given status.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes {"error": msg}.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, map[string]string{"error": msg})
}

// Auth verifies a Bearer JWT and puts the user id in the request context.
// Returns 401 when the token is missing or invalid.
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token := strings.TrimPrefix(header, "Bearer ")
			if token == "" || token == header {
				// Also accept ?token= for WebSocket upgrades.
				token = r.URL.Query().Get("token")
			}
			if token == "" {
				WriteError(w, http.StatusUnauthorized, "missing token")
				return
			}
			claims, err := auth.ValidateJWT(secret, token)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORS allows the web client (localhost:3000 by default) to call the API.
func CORS(allowOrigins string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (allowOrigins == "*" || strings.Contains(allowOrigins, origin)) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
