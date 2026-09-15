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

// AuthMiddleware validates the JWT from the Authorization header
// and stores the authenticated user's ID in the request context.
// It returns 401 Unauthorized if the token is missing, invalid, or expired.
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

// MaxBytesMiddleware protects the server from Denial of Service (DoS) attacks.
// It strictly limits the size of incoming request bodies (e.g., to 1 Megabyte).
// If a malicious user tries to upload a 500MB ticket description to crash our
// in-memory database, this cuts the connection immediately.
func MaxBytesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Limit the request body to 1 MB (1,048,576 bytes)
		r.Body = http.MaxBytesReader(w, r.Body, 1048576)
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
