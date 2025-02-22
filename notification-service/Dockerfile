FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
# Skip go mod download since there are no dependencies
COPY main.go ./
RUN go build -o notification-service

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/notification-service .
EXPOSE 3004
CMD ["./notification-service"]