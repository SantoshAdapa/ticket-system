package main

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a custom type used for context keys to avoid collisions
// with keys from other packages.
type contextKey string

// userIDKey is the context key under which the authenticated user's ID
// is stored after the JWT middleware validates their token.
const userIDKey contextKey = "user_id"

// AuthMiddleware wraps an http.Handler and enforces JWT authentication.
// It checks the Authorization header for a valid "Bearer <token>" value,
// validates the token, extracts the user ID from it, and stores the user ID
// in the request context so downstream handlers can access it.
//
// If the token is missing, malformed, or invalid, it responds with 401
// Unauthorized and stops the request from reaching the protected handler.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the Authorization header from the request.
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		// The header must follow the "Bearer <token>" format.
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		tokenString := parts[1]

		// Validate the token and extract the user ID.
		userID, err := ValidateToken(tokenString)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		// Attach the user ID to the request context so handlers can retrieve it.
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CORSMiddleware enables Cross-Origin Resource Sharing (CORS) headers on all
// HTTP responses and handles preflight OPTIONS requests cleanly. This allows
// automated web-based test runners or external browsers to make API calls seamlessly.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		// Handle browser preflight (OPTIONS) requests.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserIDFromContext extracts the authenticated user's ID from the request
// context. This should only be called inside handlers that are wrapped by
// AuthMiddleware — it will return an empty string if the middleware hasn't run.
func GetUserIDFromContext(r *http.Request) string {
	userID, ok := r.Context().Value(userIDKey).(string)
	if !ok {
		return ""
	}
	return userID
}
