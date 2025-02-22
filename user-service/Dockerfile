# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies for CGO and SQLite
RUN apk add --no-cache gcc musl-dev

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go migrations/ ./
# Explicitly enable CGO for go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -o user-service

# Runtime stage
FROM alpine:latest
# Install sqlite for runtime database initialization
RUN apk add --no-cache ca-certificates sqlite
WORKDIR /app
COPY --from=builder /app/user-service .
COPY migrations/ ./migrations/
# Ensure data directory exists and is writable
RUN mkdir -p /app/data && chmod 777 /app/data
VOLUME /app/data
EXPOSE 3001
# Initialize DB and start service, with error checking
CMD ["sh", "-c", "if [ ! -f /app/data/users.db ]; then sqlite3 /app/data/users.db < migrations/001_create_users_table.sql; fi && ./user-service"]