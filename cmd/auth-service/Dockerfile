# Stage 1: build
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build only the auth service
RUN go build -o auth-service ./cmd/auth-service

# Stage 2: runtime
FROM alpine:3.19

WORKDIR /app
RUN apk --no-cache add ca-certificates postgresql15-client

COPY --from=builder /app/auth-service .
COPY config.yaml /app/config.yaml

EXPOSE 4000

# Wait for Postgres before starting the service
CMD ["sh", "-c", "until pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER; do echo waiting for postgres; sleep 2; done; ./auth-service"]
