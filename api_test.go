package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer() *httptest.Server {
	InitJWTSecret()
	store := NewStore()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /auth/register", HandleRegister(store))
	mux.HandleFunc("POST /auth/login", HandleLogin(store))
	mux.Handle("POST /tickets", AuthMiddleware(HandleCreateTicket(store)))
	mux.Handle("GET /tickets", AuthMiddleware(HandleListTickets(store)))
	mux.Handle("GET /tickets/{id}", AuthMiddleware(HandleGetTicket(store)))
	mux.Handle("PATCH /tickets/{id}/status", AuthMiddleware(HandleUpdateTicketStatus(store)))

	return httptest.NewServer(CORSMiddleware(mux))
}

func TestEndToEndAPI(t *testing.T) {
	ts := setupTestServer()
	defer ts.Close()

	client := ts.Client()

	// 1. Health Check
	res, err := client.Get(ts.URL + "/health")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("GET /health failed: expected 200, got %v", res.StatusCode)
	}

	// 2. Register User A
	userABody, _ := json.Marshal(map[string]string{"email": "usera@example.com", "password": "password123"})
	res, err = client.Post(ts.URL+"/auth/register", "application/json", bytes.NewBuffer(userABody))
	if err != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /auth/register User A failed: expected 201, got %v", res.StatusCode)
	}

	// 2b. Register Duplicate Email -> 409 Conflict
	res, _ = client.Post(ts.URL+"/auth/register", "application/json", bytes.NewBuffer(userABody))
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("POST /auth/register duplicate failed: expected 409, got %v", res.StatusCode)
	}

	// 3. Login User A
	res, err = client.Post(ts.URL+"/auth/login", "application/json", bytes.NewBuffer(userABody))
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("POST /auth/login User A failed: expected 200, got %v", res.StatusCode)
	}
	var loginA struct{ Token string }
	json.NewDecoder(res.Body).Decode(&loginA)
	tokenA := loginA.Token

	// 4. Register & Login User B
	userBBody, _ := json.Marshal(map[string]string{"email": "userb@example.com", "password": "password123"})
	client.Post(ts.URL+"/auth/register", "application/json", bytes.NewBuffer(userBBody))
	res, _ = client.Post(ts.URL+"/auth/login", "application/json", bytes.NewBuffer(userBBody))
	var loginB struct{ Token string }
	json.NewDecoder(res.Body).Decode(&loginB)
	tokenB := loginB.Token

	// 5. Create Ticket for User A
	ticketReq, _ := json.Marshal(map[string]string{"title": "Bug in login", "description": "Cannot click submit"})
	req, _ := http.NewRequest("POST", ts.URL+"/tickets", bytes.NewBuffer(ticketReq))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	res, err = client.Do(req)
	if err != nil || res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /tickets User A failed: expected 201, got %v", res.StatusCode)
	}
	var createdTicket Ticket
	json.NewDecoder(res.Body).Decode(&createdTicket)

	// 6. List Tickets for User B -> Should be empty ([])
	req, _ = http.NewRequest("GET", ts.URL+"/tickets", nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	res, _ = client.Do(req)
	var ticketsB []Ticket
	json.NewDecoder(res.Body).Decode(&ticketsB)
	if len(ticketsB) != 0 {
		t.Fatalf("GET /tickets User B: expected 0 tickets, got %d", len(ticketsB))
	}

	// 7. Get Ticket by ID - User A (Owner) -> 200 OK
	req, _ = http.NewRequest("GET", ts.URL+"/tickets/"+createdTicket.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	res, _ = client.Do(req)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /tickets/{id} User A: expected 200, got %v", res.StatusCode)
	}

	// 8. Get Ticket by ID - User B (Non-owner) -> 403 Forbidden
	req, _ = http.NewRequest("GET", ts.URL+"/tickets/"+createdTicket.ID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenB)
	res, _ = client.Do(req)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("GET /tickets/{id} User B: expected 403 Forbidden, got %v", res.StatusCode)
	}

	// 9. Get Non-existent Ticket -> 404 Not Found
	req, _ = http.NewRequest("GET", ts.URL+"/tickets/non-existent-id", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	res, _ = client.Do(req)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /tickets/{non-existent} expected 404, got %v", res.StatusCode)
	}

	// 10. Update Status - open -> in_progress (User A) -> 200 OK
	statusReq, _ := json.Marshal(map[string]string{"status": "in_progress"})
	req, _ = http.NewRequest("PATCH", ts.URL+"/tickets/"+createdTicket.ID+"/status", bytes.NewBuffer(statusReq))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	res, _ = client.Do(req)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH status open->in_progress expected 200, got %v", res.StatusCode)
	}

	// 11. Update Status - in_progress -> open (Backward transition) -> 409 Conflict
	statusReq, _ = json.Marshal(map[string]string{"status": "open"})
	req, _ = http.NewRequest("PATCH", ts.URL+"/tickets/"+createdTicket.ID+"/status", bytes.NewBuffer(statusReq))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	req.Header.Set("Content-Type", "application/json")
	res, _ = client.Do(req)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("PATCH status in_progress->open expected 409, got %v", res.StatusCode)
	}

	// 12. Update Status - User B on User A's ticket -> 403 Forbidden
	req, _ = http.NewRequest("PATCH", ts.URL+"/tickets/"+createdTicket.ID+"/status", bytes.NewBuffer(statusReq))
	req.Header.Set("Authorization", "Bearer "+tokenB)
	req.Header.Set("Content-Type", "application/json")
	res, _ = client.Do(req)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("PATCH status User B on User A ticket expected 403, got %v", res.StatusCode)
	}
}
