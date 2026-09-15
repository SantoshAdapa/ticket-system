# ---- Build stage ----
# Use the official Go image to compile the application into a single binary.
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Copy dependency manifests first so Docker can cache this layer.
# If go.mod/go.sum don't change, Docker skips re-downloading dependencies
# on every rebuild, which speeds up iteration.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code and build a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ticket-system .

# ---- Final stage ----
# Use a minimal base image — no Go toolchain, no build tools, just the
# compiled binary. This keeps the deployed image small and reduces
# attack surface.
FROM alpine:3.20

WORKDIR /app

# Copy only the compiled binary from the build stage.
COPY --from=builder /app/ticket-system .

# Copy the static frontend files so the web server can serve them.
COPY --from=builder /app/static ./static

# The service listens on port 8080, as required by the assignment.
EXPOSE 8080

# Run the compiled binary.
CMD ["./ticket-system"]