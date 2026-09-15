package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// jwtSecret holds the key used to sign and verify JWT tokens.
// It is loaded from the JWT_SECRET environment variable at startup.
var jwtSecret []byte

// InitJWTSecret reads the JWT signing key from the JWT_SECRET environment variable.
func InitJWTSecret() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable is required")
	}
	jwtSecret = []byte(secret)
}

// hashPassword takes a plain-text password and returns its bcrypt hash.
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// checkPassword compares a plain-text password against a bcrypt hash.
func checkPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateToken creates a signed JWT containing the user's ID and a 24-hour expiration.
func generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signedToken, nil
}

// ValidateToken parses and validates a JWT string.
func ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Ensure the token was signed using the expected HMAC (HS256) method.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	// Extract the user_id claim from the token payload.
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("user_id claim missing or not a string")
	}

	return userID, nil
}

// registerRequest is the expected JSON body for the registration endpoint.
type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginRequest is the expected JSON body for the login endpoint.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// HandleRegister processes POST /auth/register requests.
// It creates a new user account with a hashed password and returns the
// user's public information (never the password or hash).
// Returns 201 on success, 400 for missing fields, 409 if the email is taken.
func HandleRegister(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest

		// Parse the JSON request body.
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		// Validate that both required fields are present and the email looks valid.
		email := strings.ToLower(strings.TrimSpace(req.Email))
		if email == "" || !strings.Contains(email, "@") || strings.TrimSpace(req.Password) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid email and password are required"})
			return
		}

		// Hash the password so we never store the plain text.
		hashedPassword, err := hashPassword(req.Password)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to process password"})
			return
		}

		// Try to create the user — this will fail if the email is already taken.
		user, created := store.CreateUser(email, hashedPassword)
		if !created {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already registered"})
			return
		}

		// Return the user's public info (the password hash is excluded by json:"-").
		writeJSON(w, http.StatusCreated, user)
	}
}

// HandleLogin processes POST /auth/login requests.
// It verifies the email and password, then returns a JWT token.
// Returns 200 with a token on success, 401 on bad credentials, 400 on
// missing fields.
func HandleLogin(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest

		// Parse the JSON request body.
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		// Validate that both fields are present.
		email := strings.ToLower(strings.TrimSpace(req.Email))
		if email == "" || strings.TrimSpace(req.Password) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
			return
		}

		// Look up the user by email.
		user, found := store.GetUserByEmail(email)
		if !found {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}

		// Verify the password against the stored hash.
		if !checkPassword(req.Password, user.PasswordHash) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}

		// Generate a JWT for the authenticated user.
		token, err := generateToken(user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"token": token})
	}
}

// writeJSON is a helper that serializes a value to JSON, sets the correct
// Content-Type header, and writes the response with the given HTTP status code.
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// Encode the data as JSON directly to the response writer.
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If encoding fails at this point, there's not much we can do since
		// headers are already sent. Log it for debugging.
		log.Printf("ERROR: failed to write JSON response: %v", err)
	}
}
