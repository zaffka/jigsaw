# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o jigsaw .

# Final stage
FROM alpine:3.23

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /app/jigsaw .

# Copy migrations (needed for embed.FS)
COPY --from=builder /app/internal/migrate/sql ./internal/migrate/sql

# Expose port (configured via PORT env var, default 8080)
EXPOSE 8080

# Run the application
CMD ["./jigsaw"]
