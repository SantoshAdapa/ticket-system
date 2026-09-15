package main

import "testing"

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   TicketStatus
		expected bool
	}{
		{"Valid open", StatusOpen, true},
		{"Valid in_progress", StatusInProgress, true},
		{"Valid closed", StatusClosed, true},
		{"Invalid empty", "", false},
		{"Invalid arbitrary string", "resolved", false},
		{"Invalid typo", "In_Progress", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidStatus(tt.status)
			if result != tt.expected {
				t.Errorf("IsValidStatus(%q) = %v; expected %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     TicketStatus
		to       TicketStatus
		expected bool
	}{
		// Valid transitions
		{"open to in_progress", StatusOpen, StatusInProgress, true},
		{"in_progress to closed", StatusInProgress, StatusClosed, true},

		// Invalid forward skips
		{"open to closed", StatusOpen, StatusClosed, false},

		// Invalid backward transitions
		{"in_progress to open", StatusInProgress, StatusOpen, false},
		{"closed to in_progress", StatusClosed, StatusInProgress, false},
		{"closed to open", StatusClosed, StatusOpen, false},

		// Invalid self transitions
		{"open to open", StatusOpen, StatusOpen, false},
		{"in_progress to in_progress", StatusInProgress, StatusInProgress, false},
		{"closed to closed", StatusClosed, StatusClosed, false},

		// Invalid arbitrary transitions
		{"invalid from", TicketStatus("unknown"), StatusOpen, false},
		{"invalid to", StatusOpen, TicketStatus("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("IsValidTransition(%q, %q) = %v; expected %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}
