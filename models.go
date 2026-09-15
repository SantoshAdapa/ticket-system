package main

import "time"

// TicketStatus represents the current state of a support ticket.
// Valid values are "open", "in_progress", and "closed".
type TicketStatus string

const (
	// StatusOpen is the default status when a ticket is first created.
	StatusOpen TicketStatus = "open"

	// StatusInProgress means someone is actively working on the ticket.
	StatusInProgress TicketStatus = "in_progress"

	// StatusClosed means the ticket has been resolved and is no longer active.
	StatusClosed TicketStatus = "closed"
)

// IsValidStatus checks whether a given string is one of the three allowed
// ticket statuses: "open", "in_progress", or "closed".
func IsValidStatus(status TicketStatus) bool {
	switch status {
	case StatusOpen, StatusInProgress, StatusClosed:
		return true
	default:
		return false
	}
}

// IsValidTransition checks whether moving from one status to another is
// allowed by the ticket state machine. The allowed transitions are:
//
//	open        -> in_progress
//	in_progress -> closed
//
// All other transitions (including staying in the same status) are rejected.
func IsValidTransition(from, to TicketStatus) bool {
	// Define every legal transition explicitly.
	allowed := map[TicketStatus][]TicketStatus{
		StatusOpen:       {StatusInProgress},
		StatusInProgress: {StatusClosed},
		// StatusClosed has no outgoing transitions — a closed ticket stays closed.
	}

	for _, target := range allowed[from] {
		if target == to {
			return true
		}
	}
	return false
}

// User represents a registered user in the system.
type User struct {
	// ID is the unique identifier for this user (UUID).
	ID string `json:"id"`

	// Email is the user's email address, used for login. Must be unique.
	Email string `json:"email"`

	// PasswordHash stores the bcrypt hash of the user's password.
	// This field is never included in JSON responses (json:"-").
	PasswordHash string `json:"-"`

	// CreatedAt records when the user account was created.
	CreatedAt time.Time `json:"created_at"`
}

// Ticket represents a support ticket created by a user.
type Ticket struct {
	// ID is the unique identifier for this ticket (UUID).
	ID string `json:"id"`

	// Title is a short summary of the ticket's subject.
	Title string `json:"title"`

	// Description provides additional detail about the issue.
	Description string `json:"description"`

	// Status is the current state: "open", "in_progress", or "closed".
	Status TicketStatus `json:"status"`

	// UserID links this ticket to the user who created it.
	UserID string `json:"user_id"`

	// CreatedAt records when the ticket was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt records when the ticket was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}
