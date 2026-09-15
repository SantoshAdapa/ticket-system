package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// createTicketRequest is the expected JSON body for creating a new ticket.
type createTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// updateStatusRequest is the expected JSON body for updating a ticket's status.
type updateStatusRequest struct {
	Status TicketStatus `json:"status"`
}

// HandleCreateTicket processes POST /tickets requests.
// It creates a new ticket owned by the authenticated user with status "open".
// Returns 201 on success, 400 if title or description is missing.
func HandleCreateTicket(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the authenticated user's ID from the request context.
		userID := GetUserIDFromContext(r)

		var req createTicketRequest

		// Parse the JSON request body.
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		// Both title and description are required.
		if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Description) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title and description are required"})
			return
		}

		// Create the ticket in the store and return it.
		ticket := store.CreateTicket(userID, strings.TrimSpace(req.Title), strings.TrimSpace(req.Description))
		writeJSON(w, http.StatusCreated, ticket)
	}
}

// HandleListTickets processes GET /tickets requests.
// It returns a JSON array of only the authenticated user's tickets.
// Always returns 200, even if the array is empty.
func HandleListTickets(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the authenticated user's ID from the request context.
		userID := GetUserIDFromContext(r)

		// Fetch all tickets belonging to this user.
		tickets := store.GetTicketsByUserID(userID)

		writeJSON(w, http.StatusOK, tickets)
	}
}

// HandleGetTicket processes GET /tickets/{id} requests.
// It returns a single ticket if it exists and is owned by the authenticated user.
// Returns 200 if found and owned, 404 if the ticket ID doesn't exist,
// and 403 if the ticket exists but belongs to a different user.
func HandleGetTicket(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the authenticated user's ID from the request context.
		userID := GetUserIDFromContext(r)

		// Extract the ticket ID from the URL path parameter.
		ticketID := r.PathValue("id")

		// Look up the ticket.
		ticket, found := store.GetTicketByID(ticketID)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ticket not found"})
			return
		}

		// Check that the authenticated user owns this ticket.
		if ticket.UserID != userID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		writeJSON(w, http.StatusOK, ticket)
	}
}

// HandleUpdateTicketStatus processes PATCH /tickets/{id}/status requests.
// It updates a ticket's status if the caller owns the ticket and the
// status transition is valid according to the state machine.
//
// Returns:
//   - 200 on success
//   - 400 if the new status is not a valid status value
//   - 409 if the transition is not allowed (e.g. closed -> anything)
//   - 404 if the ticket ID doesn't exist
//   - 403 if the ticket belongs to another user
func HandleUpdateTicketStatus(store *Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the authenticated user's ID from the request context.
		userID := GetUserIDFromContext(r)

		// Extract the ticket ID from the URL path parameter.
		ticketID := r.PathValue("id")

		var req updateStatusRequest

		// Parse the JSON request body.
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		// Look up the ticket to check existence and ownership first.
		ticket, found := store.GetTicketByID(ticketID)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ticket not found"})
			return
		}

		// Check that the authenticated user owns this ticket before inspecting parameters.
		if ticket.UserID != userID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		// Validate that the provided status is one of the three allowed values.
		if !IsValidStatus(req.Status) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status value"})
			return
		}

		// Check that the requested status transition is allowed by the state machine.
		if !IsValidTransition(ticket.Status, req.Status) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "status transition not allowed"})
			return
		}

		// Apply the status update in the store.
		updatedTicket, ok := store.UpdateTicketStatus(ticketID, req.Status)
		if !ok {
			// This shouldn't happen since we just confirmed the ticket exists,
			// but handle it defensively.
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ticket not found"})
			return
		}

		writeJSON(w, http.StatusOK, updatedTicket)
	}
}
