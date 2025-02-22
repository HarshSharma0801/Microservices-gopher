FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN go build -o payment-service

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/payment-service .
EXPOSE 3003
CMD ["./payment-service"]