package main

import (
	"log"
	"net/http"
)

func main() {
	// Load the JWT signing secret from the environment (or fall back to a
	// development default with a warning).
	InitJWTSecret()

	// Create the in-memory data store that holds users and tickets.
	store := NewStore()

	// Set up the HTTP router using Go 1.22+ method+path patterns.
	mux := http.NewServeMux()

	// --- Public routes (no authentication required) ---

	// Health check endpoint — used by load balancers and uptime monitors.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// User registration — creates a new account.
	mux.HandleFunc("POST /auth/register", HandleRegister(store))

	// User login — returns a JWT token for authenticated access.
	mux.HandleFunc("POST /auth/login", HandleLogin(store))

	// --- Protected routes (require valid JWT in Authorization header) ---

	// Create a new ticket.
	mux.Handle("POST /tickets", AuthMiddleware(HandleCreateTicket(store)))

	// List all tickets belonging to the authenticated user.
	mux.Handle("GET /tickets", AuthMiddleware(HandleListTickets(store)))

	// Get a single ticket by ID (must be owned by the authenticated user).
	mux.Handle("GET /tickets/{id}", AuthMiddleware(HandleGetTicket(store)))

	// Update a ticket's status (must be owned by the authenticated user).
	mux.Handle("PATCH /tickets/{id}/status", AuthMiddleware(HandleUpdateTicketStatus(store)))

	// Start the HTTP server on port 8080.
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("FATAL: server failed to start: %v", err)
	}
}
