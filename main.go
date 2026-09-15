package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Load the JWT signing secret from the environment (or fall back to a
	// development default with a warning).
	InitJWTSecret()

	// Create the in-memory data store that holds users and tickets.
	store := NewStore()

	// Initialize the HTTP router and register all endpoints.
	mux := http.NewServeMux()

	// --- Public routes (no authentication required) ---

	// The health check endpoint allows load balancers to verify the service is running.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// User registration and login
	mux.HandleFunc("POST /auth/register", HandleRegister(store))
	mux.HandleFunc("POST /auth/login", HandleLogin(store))

	// Serve the frontend UI files from the static directory.
	mux.Handle("GET /", http.FileServer(http.Dir("./static")))

	// --- Protected routes (require JWT) ---
	
	// Create a new ticket
	mux.Handle("POST /tickets", AuthMiddleware(HandleCreateTicket(store)))
	
	// List all tickets owned by the current user
	mux.Handle("GET /tickets", AuthMiddleware(HandleListTickets(store)))
	
	// Get a specific ticket (must be owned by the current user)
	mux.Handle("GET /tickets/{id}", AuthMiddleware(HandleGetTicket(store)))
	
	// Update the status of a specific ticket
	mux.Handle("PATCH /tickets/{id}/status", AuthMiddleware(HandleUpdateTicketStatus(store)))

	// Get the port from the environment variable (Render sets this automatically).
	// If it's empty, default to 8080 for local development.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on :%s", port)
	
	// Wrap the entire router in the MaxBytesMiddleware size limit
	handler := MaxBytesMiddleware(mux)
	
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("FATAL: server failed to start: %v", err)
	}
}
