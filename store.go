package main

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Store holds all application data in memory using Go maps.
// All methods are safe to call from multiple goroutines because
// every read/write is protected by a read-write mutex.
type Store struct {
	// mu protects all maps below from concurrent access.
	mu sync.RWMutex

	// users maps user ID -> User.
	users map[string]User

	// emailToUserID maps email -> user ID for fast duplicate-email checks and login lookups.
	emailToUserID map[string]string

	// tickets maps ticket ID -> Ticket.
	tickets map[string]Ticket

	// userTickets maps user ID -> list of ticket IDs that user owns.
	userTickets map[string][]string
}

// NewStore creates and returns a fresh, empty in-memory store.
func NewStore() *Store {
	return &Store{
		users:         make(map[string]User),
		emailToUserID: make(map[string]string),
		tickets:       make(map[string]Ticket),
		userTickets:   make(map[string][]string),
	}
}

// CreateUser adds a new user to the store. It returns the created User and
// a boolean indicating success. If the email is already taken, it returns
// an empty User and false.
func (s *Store) CreateUser(email, passwordHash string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check whether this email address is already registered.
	if _, exists := s.emailToUserID[email]; exists {
		return User{}, false
	}

	user := User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}

	// Store the user and the email-to-ID mapping.
	s.users[user.ID] = user
	s.emailToUserID[email] = user.ID
	return user, true
}

// GetUserByEmail looks up a user by their email address.
// Returns the User and true if found, or an empty User and false if not.
func (s *Store) GetUserByEmail(email string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userID, exists := s.emailToUserID[email]
	if !exists {
		return User{}, false
	}
	user := s.users[userID]
	return user, true
}

// CreateTicket adds a new ticket to the store, owned by the given user.
// It returns the newly created Ticket.
func (s *Store) CreateTicket(userID, title, description string) Ticket {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ticket := Ticket{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		Status:      StatusOpen,
		UserID:      userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store the ticket and record it in the user's ticket list.
	s.tickets[ticket.ID] = ticket
	s.userTickets[userID] = append(s.userTickets[userID], ticket.ID)
	return ticket
}

// GetTicketByID retrieves a single ticket by its ID.
// Returns the Ticket and true if found, or an empty Ticket and false if not.
func (s *Store) GetTicketByID(ticketID string) (Ticket, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ticket, exists := s.tickets[ticketID]
	return ticket, exists
}

// GetTicketsByUserID returns all tickets belonging to a specific user.
// If the user has no tickets, an empty slice is returned (never nil).
func (s *Store) GetTicketsByUserID(userID string) []Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ticketIDs := s.userTickets[userID]
	result := make([]Ticket, 0, len(ticketIDs))
	for _, id := range ticketIDs {
		result = append(result, s.tickets[id])
	}
	return result
}

// UpdateTicketStatus changes a ticket's status and updates its UpdatedAt
// timestamp. Returns the updated Ticket and true on success, or an empty
// Ticket and false if the ticket ID does not exist.
func (s *Store) UpdateTicketStatus(ticketID string, newStatus TicketStatus) (Ticket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ticket, exists := s.tickets[ticketID]
	if !exists {
		return Ticket{}, false
	}

	ticket.Status = newStatus
	ticket.UpdatedAt = time.Now().UTC()
	s.tickets[ticketID] = ticket
	return ticket, true
}
