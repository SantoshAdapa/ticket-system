# Ticket System

A small backend service for a support ticket system, built in Go. Users can register, log in, create tickets, view only their own tickets, and update the status of their own tickets. It also includes a lightweight vanilla JS frontend.

## Live Deployment

- **Deployed URL:** https://ticket-system-e3ku.onrender.com (Serves the interactive frontend UI)
- **Public health check:** https://ticket-system-e3ku.onrender.com/health

Note: this is hosted on Render's free tier, which spins the service down after periods of inactivity. The first request after idle time may take 30-60 seconds while the service restarts. Subsequent requests are fast.

## Tech Stack

- **Language:** Go (standard library `net/http` only — no web framework)
- **Frontend:** Vanilla HTML/CSS/JS (served directly from the Go binary)
- **Auth:** JWT (HS256), via `github.com/golang-jwt/jwt/v5`
- **Password hashing:** bcrypt, via `golang.org/x/crypto/bcrypt`
- **Storage:** in-memory (Go maps protected by a mutex) — no external database
- **IDs:** UUIDs, via `github.com/google/uuid`

## Project Structure

| File | Purpose |
|---|---|
| `models.go` | User and Ticket data structures, ticket status state machine |
| `store.go` | Thread-safe in-memory data store |
| `auth.go` | Password hashing, JWT creation/validation, register and login handlers |
| `middleware.go` | JWT authentication and CORS middleware |
| `tickets.go` | Ticket create, list, get, and status update handlers |
| `main.go` | Route registration, static file serving, and server startup |
| `static/` | Vanilla JS frontend files (`index.html`, `style.css`, `app.js`) |
| `*_test.go` | Comprehensive Unit and HTTP End-to-End integration tests |

## API Endpoints

| Method | Endpoint | Auth Required | Purpose |
|---|---|---|---|
| GET | `/health` | No | Health check |
| POST | `/auth/register` | No | Register a new user |
| POST | `/auth/login` | No | Log in, returns a JWT |
| POST | `/tickets` | Yes | Create a ticket |
| GET | `/tickets` | Yes | List the logged-in user's own tickets |
| GET | `/tickets/{id}` | Yes | Get a single ticket (must be owned by the caller) |
| PATCH | `/tickets/{id}/status` | Yes | Update a ticket's status (must be owned by the caller) |

Protected endpoints require an `Authorization: Bearer <token>` header, using the token returned by `/auth/login`. Cross-Origin Resource Sharing (CORS) is enabled.

### Ticket status flow

```
open -> in_progress -> closed
```

Statuses can only move forward one step at a time. A closed ticket cannot be reopened or moved to any other status.

## Automated Tests

This project includes a comprehensive end-to-end integration test suite that tests all APIs, edge cases, CORS, and ownership validations.

Run the tests via:
```bash
go test -v ./...
```

## Running Locally (without Docker)

Requires Go 1.23 or later installed.

```bash
go mod download
$env:JWT_SECRET = "your-secret-here"   # PowerShell
go run .
```

The server starts on port 8080 and serves the UI at `http://localhost:8080`. Verify the API is running:

```bash
curl http://localhost:8080/health
```

If `JWT_SECRET` is not set, the application falls back to a hardcoded development-only secret and prints a clear warning to the logs. This should never be relied on outside of quick local testing.

## Running Locally with Docker

```bash
docker build -t ticket-system .
docker run -p 8080:8080 -e JWT_SECRET=your-secret-here ticket-system
```

Navigate to `http://localhost:8080` in your browser to interact with the ticket system UI.

## Environment Variables

See `.env.example`.

| Variable | Required | Description |
|---|---|---|
| `JWT_SECRET` | Recommended | Secret key used to sign JWT tokens. Falls back to an insecure default (with a logged warning) if unset. |
| `PORT` | Optional | Port the server listens on (defaults to 8080). Used by Render for dynamic port binding. |

## Assumptions Made

- **Storage is in-memory only**, as permitted by the assignment's scope section. This means all data (users and tickets) is lost whenever the server process restarts. On the deployed free-tier host, this can happen if the service is redeployed or, in some cases, when it spins back up after an idle period — this is a deliberate trade-off for simplicity, matching the assignment's instruction not to over-engineer.
- **Ticket ownership field is named `user_id`** in JSON responses. The assignment did not specify an exact field name for this, so a reasonable name was chosen and used consistently across all endpoints.
- **Status transitions are strictly sequential**: `open -> in_progress` and `in_progress -> closed` are the only two allowed transitions. Skipping directly from `open` to `closed`, or re-submitting the same status a ticket is already in, is treated as an invalid transition (`409 Conflict`).
- **Email is Case-Insensitive**: User emails are normalized to lowercase during registration and login to prevent duplicate registrations and login errors with mismatched casing (e.g. `User@domain.com` vs `user@domain.com`).
- **Strict Authorization Boundary**: Requesting a ticket that exists but is owned by a different user returns `403 Forbidden`, regardless of the operation. Requesting a nonexistent ticket returns `404 Not Found`. Payload parameters (like an invalid status string) are validated *after* ensuring the user actually owns the ticket, to prevent authorization leaks.
- **Login failure messages are intentionally generic** (`"invalid email or password"`) regardless of whether the email doesn't exist or the password is wrong, to avoid revealing which one was incorrect.